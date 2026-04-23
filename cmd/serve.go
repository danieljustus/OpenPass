package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"

	"github.com/danieljustus/OpenPass/internal/mcp"
	"github.com/danieljustus/OpenPass/internal/metrics"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var serveSignalNotify = signal.Notify

// Version is set at build time via -ldflags. Defaults to "dev".
var Version = "dev"

var mcpFactory = struct {
	New func(vault *vaultpkg.Vault, agentName string, transport string) (*mcp.Server, error)
}{
	New: mcp.New,
}

var runStdioServerFunc = runStdioServer
var runHTTPServerFunc = runHTTPServer
var serveUnlockVault = unlockVault

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for agent access",
	Long: `Start an MCP server that exposes vault operations to AI agents.

Each agent must be configured in ~/.openpass/config.yaml with specific
permissions and scope restrictions.

The server can run in HTTP mode or stdio mode.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName, err := cmd.Flags().GetString("agent")
		if err != nil {
			return fmt.Errorf("read agent flag: %w", err)
		}

		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return fmt.Errorf("read port flag: %w", err)
		}

		stdioFlag, err := cmd.Flags().GetBool("stdio")
		if err != nil {
			return fmt.Errorf("read stdio flag: %w", err)
		}

		bind, err := cmd.Flags().GetString("bind")
		if err != nil {
			return fmt.Errorf("read bind flag: %w", err)
		}

		if bind == "" {
			return fmt.Errorf("--bind must not be empty; use '127.0.0.1' for localhost-only")
		}

		// --agent is required for stdio (agent is fixed at startup)
		// HTTP mode resolves agents per-request via X-OpenPass-Agent header
		if stdioFlag && agentName == "" {
			return fmt.Errorf("--agent is required in --stdio mode")
		}

		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		if !vaultpkg.IsInitialized(vaultDir) {
			return fmt.Errorf("vault not initialized. Run 'openpass init' first")
		}

		// Unlock vault for HTTP mode (always needed) or stdio with an agent
		var vault *vaultpkg.Vault
		if agentName != "" || !stdioFlag {
			vault, err = serveUnlockVault(vaultDir, !stdioFlag)
			if err != nil {
				return err
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		serveSignalNotify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		go func() {
			sig := <-sigCh
			if sig == syscall.SIGQUIT {
				fmt.Fprintln(os.Stderr, "Received SIGQUIT, shutting down gracefully...")
			}
			cancel()
		}()

		var wg sync.WaitGroup
		errCh := make(chan error, 2)

		if stdioFlag {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := runStdioServerFunc(ctx, vault, agentName); err != nil {
					errCh <- fmt.Errorf("stdio server: %w", err)
				}
			}()
		}

		if !stdioFlag {
			actualPort, isPreferred, err := findAvailablePort(bind, port)
			if err != nil {
				return fmt.Errorf("port allocation failed: %w", err)
			}
			if !isPreferred {
				fmt.Fprintf(os.Stderr, "Port %d is in use, using port %d instead\n", port, actualPort)
			}
			if err := saveRuntimePort(vaultDir, actualPort); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save runtime port: %v\n", err)
			}
			fmt.Fprintf(os.Stderr, "MCP server listening on %s:%d\n", bind, actualPort)

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := runHTTPServerFunc(ctx, bind, actualPort, vault); err != nil {
					errCh <- fmt.Errorf("http server: %w", err)
				}
			}()
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			return nil
		case err := <-errCh:
			cancel()
			return err
		case <-ctx.Done():
			if vDir, err := vaultPath(); err == nil {
				_ = clearRuntimePort(vDir)
			}
			return nil
		}
	},
}

func runStdioServer(ctx context.Context, vault *vaultpkg.Vault, agentName string) error {
	var mcpServer *mcp.Server
	if vault != nil && agentName != "" {
		var err error
		mcpServer, err = mcpFactory.New(vault, agentName, "stdio")
		if err != nil {
			return fmt.Errorf("failed to create MCP server: %w", err)
		}
		defer func() { _ = mcpServer.Close() }()
	}

	transport := mcp.NewStdioTransport()
	handler := mcp.NewProtocolHandler("openpass", "1.0.0", mcpServer)

	errCh := make(chan error, 1)
	go func() {
		errCh <- transport.Start(ctx, handler.HandleMessage)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Graceful shutdown: wait for transport to finish or timeout
		select {
		case err := <-errCh:
			return err
		case <-time.After(2 * time.Second):
			return fmt.Errorf("stdio server shutdown timeout")
		}
	}
}

func runHTTPServer(ctx context.Context, bind string, port int, v *vaultpkg.Vault) error {
	addr := fmt.Sprintf("%s:%d", bind, port)

	// Resolve token path from config or use default
	vaultDir, err := vaultPath()
	if err != nil {
		return err
	}
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

	rateLimiter := mcp.NewRateLimiter(60, time.Minute)
	stopCleanup := rateLimiter.StartCleanup(ctx, 5*time.Minute)

	// Per-agent protocol handler cache: one handler per agent name, created on first request.
	// Each handler holds its own MCP server and audit log file handle.
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
			mcpServer, err := mcpFactory.New(v, agentName, "http")
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

	// Health endpoint — no auth required
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"status":    "healthy",
			"port":      port,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   Version,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Metrics endpoint — Prometheus format, no auth required
	mux.Handle("/metrics", promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{}))

	// MCP endpoint — bearer auth + agent header, JSON-RPC via POST
	const maxRequestBodySize = 1 * 1024 * 1024
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeMCPHTTPError(w, http.StatusMethodNotAllowed, nil, mcp.ErrCodeInvalidRequest, "method not allowed")
			return
		}
		if !isJSONContentType(r.Header.Get("Content-Type")) {
			writeMCPHTTPError(w, http.StatusUnsupportedMediaType, nil, mcp.ErrCodeInvalidRequest, "Content-Type must be application/json")
			return
		}
		if !acceptsMCPHTTPResponse(r.Header.Values("Accept")) {
			writeMCPHTTPError(w, http.StatusNotAcceptable, nil, mcp.ErrCodeInvalidRequest, "Accept must include application/json and text/event-stream")
			return
		}

		var msg mcp.Message
		bodyReader := http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		if err := json.NewDecoder(bodyReader).Decode(&msg); err != nil {
			if err.Error() == "http: request body too large" {
				writeMCPHTTPError(w, http.StatusRequestEntityTooLarge, nil, mcp.ErrCodeParseError, "request body too large")
				return
			}
			writeMCPHTTPError(w, http.StatusBadRequest, nil, mcp.ErrCodeParseError, "invalid JSON")
			return
		}

		protocolVersion := strings.TrimSpace(r.Header.Get("MCP-Protocol-Version"))
		if protocolVersion == "" && msg.Method != "initialize" {
			protocolVersion = mcp.DefaultHTTPProtocolVersion
		}
		if protocolVersion != "" && !mcp.IsSupportedProtocolVersion(protocolVersion) {
			writeMCPHTTPError(w, http.StatusBadRequest, msg.ID, mcp.ErrCodeInvalidRequest, "unsupported MCP-Protocol-Version")
			return
		}

		// Resolve agent from X-OpenPass-Agent header (set by AgentHeaderMiddleware)
		agentName := mcp.AgentFromContext(r.Context())
		handler, err := handlerForAgent(agentName)
		if err != nil {
			writeMCPHTTPError(w, http.StatusForbidden, msg.ID, mcp.ErrCodeInternalError, err.Error())
			return
		}

		resp, err := handler.HandleMessage(r.Context(), &msg)
		if err != nil {
			writeMCPHTTPError(w, http.StatusInternalServerError, msg.ID, mcp.ErrCodeInternalError, err.Error())
			return
		}
		if resp == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.Handle("/mcp", mcp.RateLimiterMiddleware(rateLimiter, mcp.OriginValidationMiddleware(addr, mcp.BearerAuthMiddleware(token, mcp.AgentHeaderMiddleware(mcpHandler)))))

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
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	//nolint:contextcheck // Shutdown goroutine cleans up cached handlers when ctx is canceled - no request-scoped context needed
	go func() {
		<-ctx.Done()
		if stopCleanup != nil {
			stopCleanup()
		}
		_ = rateLimiter.Close()
		// Close all cached agent protocol handlers (closes audit log file handles)
		cacheMu.Lock()
		for _, h := range handlerCache {
			_ = h.Close()
		}
		cacheMu.Unlock()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func writeMCPHTTPError(w http.ResponseWriter, status int, id json.RawMessage, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(mcp.NewErrorResponse(id, code, message, nil))
}

func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

func acceptsMCPHTTPResponse(values []string) bool {
	acceptsJSON := false
	acceptsSSE := false
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(part))
			if err != nil {
				continue
			}
			switch mediaType {
			case "*/*", "application/*", "application/json":
				acceptsJSON = true
			case "text/event-stream":
				acceptsSSE = true
			}
		}
	}
	return acceptsJSON && acceptsSSE
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().String("agent", "", "Agent name (required for --stdio; HTTP mode resolves agents per-request via X-OpenPass-Agent header)")
	serveCmd.Flags().Int("port", 8080, "Server port")
	serveCmd.Flags().Bool("stdio", false, "Enable stdio transport (for MCP)")
	serveCmd.Flags().String("bind", "127.0.0.1", "Bind address for HTTP server")
}
