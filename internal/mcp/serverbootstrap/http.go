package serverbootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/danieljustus/OpenPass/internal/audit"
	"github.com/danieljustus/OpenPass/internal/mcp"
	"github.com/danieljustus/OpenPass/internal/metrics"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

// RunHTTPServer starts the HTTP MCP server.
func RunHTTPServer(ctx context.Context, bind string, port int, v *vaultpkg.Vault, vaultDir string, version string, factory func(*vaultpkg.Vault, string, string) (*mcp.Server, error)) error {
	addr := fmt.Sprintf("%s:%d", bind, port)

	tokenPath := filepath.Join(vaultDir, "mcp-token")
	if v != nil && v.Config != nil && v.Config.MCP != nil {
		tf := v.Config.MCP.HTTPTokenFile
		if tf != "" && tf != "auto" {
			tokenPath = tf
		}
	}

	token, err := mcp.LoadOrCreateToken(tokenPath)
	if err != nil {
		return fmt.Errorf("load token: %w", err)
	}

	authAuditLog, err := audit.New("auth-failures", vaultDir)
	if err != nil {
		return fmt.Errorf("create auth audit logger: %w", err)
	}
	defer authAuditLog.Close()

	rateLimit := 60
	var trustedProxyIPs []string
	if v != nil && v.Config != nil && v.Config.MCP != nil {
		if v.Config.MCP.RateLimit >= 0 {
			rateLimit = v.Config.MCP.RateLimit
		}
		trustedProxyIPs = v.Config.MCP.TrustedProxyIPs
	}
	var rateLimiter *mcp.RateLimiter
	var stopCleanup func()
	if rateLimit > 0 {
		rateLimiter = mcp.NewRateLimiter(rateLimit, time.Minute, trustedProxyIPs...)
		stopCleanup = rateLimiter.StartCleanup(ctx, 5*time.Minute)
	}

	handlerCache := make(map[string]*mcp.ProtocolHandler)
	var cacheMu sync.RWMutex

	handlerForAgent := func(agentName string) (*mcp.ProtocolHandler, error) {
		cacheMu.RLock()
		if h, ok := handlerCache[agentName]; ok {
			cacheMu.RUnlock()
			return h, nil
		}
		cacheMu.RUnlock()

		type result struct {
			handler *mcp.ProtocolHandler
			err     error
		}
		resultChan := make(chan result, 1)

		go func() {
			mcpServer, err := factory(v, agentName, "http")
			if err != nil {
				resultChan <- result{err: err}
				return
			}
			h := mcp.NewProtocolHandler("openpass", "1.0.0", mcpServer)

			cacheMu.Lock()
			if existing, ok := handlerCache[agentName]; ok {
				_ = h.Close()
				cacheMu.Unlock()
				resultChan <- result{handler: existing}
				return
			}
			handlerCache[agentName] = h
			cacheMu.Unlock()
			resultChan <- result{handler: h}
		}()

		select {
		case res := <-resultChan:
			return res.handler, res.err
		case <-time.After(10 * time.Second):
			return nil, fmt.Errorf("handler creation timeout for agent %q: creation took longer than 10s", agentName)
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"status":    "healthy",
			"port":      port,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   version,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	metricsHandler := promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{})
	metricsAuthRequired := true
	if v != nil && v.Config != nil && v.Config.MCP != nil {
		metricsAuthRequired = v.Config.MCP.MetricsAuthRequired
	}
	if !mcp.IsLoopbackBind(bind) && metricsAuthRequired {
		mux.Handle("/metrics", mcp.BearerAuthMiddleware(token, authAuditLog, metricsHandler))
	} else {
		mux.Handle("/metrics", metricsHandler)
	}

	const maxRequestBodySize = 1 * 1024 * 1024
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			mcp.WriteMCPHTTPError(w, http.StatusMethodNotAllowed, nil, mcp.ErrCodeInvalidRequest, "method not allowed")
			return
		}
		if !mcp.IsJSONContentType(r.Header.Get("Content-Type")) {
			mcp.WriteMCPHTTPError(w, http.StatusUnsupportedMediaType, nil, mcp.ErrCodeInvalidRequest, "Content-Type must be application/json")
			return
		}
		if !mcp.AcceptsMCPHTTPResponse(r.Header.Values("Accept")) {
			mcp.WriteMCPHTTPError(w, http.StatusNotAcceptable, nil, mcp.ErrCodeInvalidRequest, "Accept must include application/json and text/event-stream")
			return
		}

		var msg mcp.Message
		bodyReader := http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		if err := json.NewDecoder(bodyReader).Decode(&msg); err != nil {
			if err.Error() == "http: request body too large" {
				mcp.WriteMCPHTTPError(w, http.StatusRequestEntityTooLarge, nil, mcp.ErrCodeParseError, "request body too large")
				return
			}
			mcp.WriteMCPHTTPError(w, http.StatusBadRequest, nil, mcp.ErrCodeParseError, "invalid JSON")
			return
		}

		protocolVersion := strings.TrimSpace(r.Header.Get("MCP-Protocol-Version"))
		if protocolVersion == "" && msg.Method != "initialize" {
			protocolVersion = mcp.DefaultHTTPProtocolVersion
		}
		if protocolVersion != "" && !mcp.IsSupportedProtocolVersion(protocolVersion) {
			mcp.WriteMCPHTTPError(w, http.StatusBadRequest, msg.ID, mcp.ErrCodeInvalidRequest, "unsupported MCP-Protocol-Version")
			return
		}

		agentName := mcp.AgentFromContext(r.Context())
		handler, err := handlerForAgent(agentName)
		if err != nil {
			mcp.WriteMCPHTTPError(w, http.StatusForbidden, msg.ID, mcp.ErrCodeInternalError, err.Error())
			return
		}

		resp, err := handler.HandleMessage(r.Context(), &msg)
		if err != nil {
			mcp.WriteMCPHTTPError(w, http.StatusInternalServerError, msg.ID, mcp.ErrCodeInternalError, err.Error())
			return
		}
		if resp == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mcpChain := mcp.OriginValidationMiddleware(addr, mcp.BearerAuthMiddleware(token, authAuditLog, mcp.AgentHeaderMiddleware(mcpHandler)))
	if rateLimiter != nil {
		mcpChain = mcp.RateLimiterMiddleware(rateLimiter, mcpChain)
	}
	mux.Handle("/mcp", mcpChain)

	readHeaderTimeout := 5 * time.Second
	readTimeout := 10 * time.Second
	writeTimeout := 10 * time.Second
	shutdownTimeout := 5 * time.Second
	if v != nil && v.Config != nil && v.Config.MCP != nil {
		mcpCfg := v.Config.MCP
		if mcpCfg.ReadHeaderTimeout > 0 {
			readHeaderTimeout = mcpCfg.ReadHeaderTimeout
		}
		if mcpCfg.ReadTimeout > 0 {
			readTimeout = mcpCfg.ReadTimeout
		}
		if mcpCfg.WriteTimeout > 0 {
			writeTimeout = mcpCfg.WriteTimeout
		}
		if mcpCfg.ShutdownTimeout > 0 {
			shutdownTimeout = mcpCfg.ShutdownTimeout
		}
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		<-ctx.Done()
		if stopCleanup != nil {
			stopCleanup()
		}
		if rateLimiter != nil {
			_ = rateLimiter.Close()
		}
		cacheMu.Lock()
		for _, h := range handlerCache {
			_ = h.Close()
		}
		cacheMu.Unlock()
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
