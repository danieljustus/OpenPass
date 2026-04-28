package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/mcp"
	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/session"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func newTestHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port //nolint:errcheck // net.Listener.Addr() does not return an error
	_ = l.Close()
	return port
}

func newTestVault(t *testing.T) *vaultpkg.Vault {
	t.Helper()
	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	if vaultFlag != nil {
		vaultFlag.Changed = false
	}

	_, err := vaultpkg.InitWithPassphrase(tmpDir, "test-passphrase", config.Default())
	if err != nil {
		t.Fatalf("init vault: %v", err)
	}

	v, err := vaultpkg.OpenWithPassphrase(tmpDir, "test-passphrase")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	return v
}

func runHTTPServerAsyncWithBind(ctx context.Context, t *testing.T, bind string, port int, v *vaultpkg.Vault) func() {
	t.Helper()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runHTTPServer(ctx, bind, port, v); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("runHTTPServer error: %v", err)
		}
	}()
	time.Sleep(200 * time.Millisecond)
	return wg.Wait
}

func runHTTPServerAsync(ctx context.Context, t *testing.T, port int, v *vaultpkg.Vault) func() {
	return runHTTPServerAsyncWithBind(ctx, t, "127.0.0.1", port, v)
}

func testMCPToken(t *testing.T) string {
	t.Helper()
	vaultDir, _ := vaultPath()
	tokenBytes, err := os.ReadFile(filepath.Join(vaultDir, "mcp-token"))
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	return strings.TrimSpace(string(tokenBytes))
}

func setValidMCPHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-OpenPass-Agent", "default")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
}

func TestRunHTTPServer_HealthEndpoint(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	client := newTestHTTPClient()
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("health Content-Type = %q, want application/json", contentType)
	}

	var healthResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if healthResp["status"] != "healthy" {
		t.Errorf("health status = %v, want healthy", healthResp["status"])
	}
	if healthResp["version"] == "" {
		t.Error("health version is empty")
	}
	if healthResp["timestamp"] == "" {
		t.Error("health timestamp is empty")
	}
	if healthResp["port"] == nil {
		t.Error("health port is missing")
	}
}

func TestRunHTTPServer_MetricsEndpoint(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	// Record some metrics to ensure they appear in the output
	metrics.RecordMCPRequest("test_tool", "test_agent", "success", 100*time.Millisecond)
	metrics.RecordAuthDenial("test_reason", "test_agent")
	metrics.RecordApproval("test_agent", "granted")
	metrics.RecordVaultOperation("test_operation", "success")

	client := newTestHTTPClient()
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
	if err != nil {
		t.Fatalf("metrics request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("metrics status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("metrics Content-Type = %q, want text/plain", contentType)
	}

	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(resp.Body)
	bodyStr := body.String()

	// Check for Go runtime metrics (always present)
	if !strings.Contains(bodyStr, "go_goroutines") {
		t.Error("metrics response missing go_goroutines")
	}
	if !strings.Contains(bodyStr, "process_cpu_seconds_total") {
		t.Error("metrics response missing process_cpu_seconds_total")
	}

	// Check that OpenPass metrics are present
	if !strings.Contains(bodyStr, "openpass_mcp_requests_total") {
		t.Error("metrics response missing openpass_mcp_requests_total")
	}
	if !strings.Contains(bodyStr, "openpass_mcp_request_duration_seconds") {
		t.Error("metrics response missing openpass_mcp_request_duration_seconds")
	}
	if !strings.Contains(bodyStr, "openpass_mcp_auth_denials_total") {
		t.Error("metrics response missing openpass_mcp_auth_denials_total")
	}
	if !strings.Contains(bodyStr, "openpass_mcp_approvals_total") {
		t.Error("metrics response missing openpass_mcp_approvals_total")
	}
	if !strings.Contains(bodyStr, "openpass_vault_operations_total") {
		t.Error("metrics response missing openpass_vault_operations_total")
	}
}

func TestRunHTTPServer_MCPEndpoint_Auth(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	vaultDir, _ := vaultPath()
	tokenBytes, err := os.ReadFile(filepath.Join(vaultDir, "mcp-token"))
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	resp, err := newTestHTTPClient().Post(baseURL+"/mcp", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("missing auth status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("missing agent header status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestRunHTTPServer_MCPEndpoint_WithAgent(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	vaultDir, _ := vaultPath()
	tokenBytes, err := os.ReadFile(filepath.Join(vaultDir, "mcp-token"))
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
		},
	}
	payload, _ := json.Marshal(msg)

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-OpenPass-Agent", "default")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should get some response (initialize handler returns a result even with nil server)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("mcp status = %d, want %d or %d", resp.StatusCode, http.StatusOK, http.StatusInternalServerError)
	}
}

func TestRunHTTPServer_MethodNotAllowed(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	vaultDir, _ := vaultPath()
	tokenBytes, err := os.ReadFile(filepath.Join(vaultDir, "mcp-token"))
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	req, _ := http.NewRequest(http.MethodGet, baseURL+"/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-OpenPass-Agent", "default")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("mcp GET request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /mcp status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestRunHTTPServer_InvalidJSON(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	vaultDir, _ := vaultPath()
	tokenBytes, err := os.ReadFile(filepath.Join(vaultDir, "mcp-token"))
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader("{broken"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-OpenPass-Agent", "default")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid JSON status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var errResp mcp.Message
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if errResp.Error == nil {
		t.Fatal("expected error in response")
	}
	if errResp.Error.Code != mcp.ErrCodeParseError {
		t.Errorf("error code = %d, want %d", errResp.Error.Code, mcp.ErrCodeParseError)
	}
	if !strings.Contains(errResp.Error.Message, "invalid JSON") {
		t.Errorf("error message = %q, want contains 'invalid JSON'", errResp.Error.Message)
	}
}

func TestRunHTTPServer_HTTPTransportHeaderValidation(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	token := testMCPToken(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	payload := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{}}}`)

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", payload)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-OpenPass-Agent", "default")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("missing content-type request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("missing Content-Type status = %d, want %d", resp.StatusCode, http.StatusUnsupportedMediaType)
	}

	req, _ = http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-OpenPass-Agent", "default")
	req.Header.Set("Content-Type", "application/json")
	resp, err = newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("missing accept request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("missing Accept status = %d, want %d", resp.StatusCode, http.StatusNotAcceptable)
	}
}

func TestRunHTTPServer_NotificationReturnsAccepted(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	token := testMCPToken(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	setValidMCPHeaders(req, token)
	req.Header.Set("MCP-Protocol-Version", "2025-11-25")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("notification request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("notification status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}
	body := new(bytes.Buffer)
	_, _ = body.ReadFrom(resp.Body)
	if strings.TrimSpace(body.String()) != "" {
		t.Fatalf("notification response body = %q, want empty", body.String())
	}
}

func TestRunHTTPServer_BadOriginForbidden(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	token := testMCPToken(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	setValidMCPHeaders(req, token)
	req.Header.Set("Origin", "https://evil.example")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("bad origin request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bad Origin status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestRunHTTPServer_UnsupportedProtocolHeader(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	token := testMCPToken(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	setValidMCPHeaders(req, token)
	req.Header.Set("MCP-Protocol-Version", "1999-01-01")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("unsupported protocol request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unsupported protocol status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestRunHTTPServer_HandlerCreationError(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	origNew := mcpFactory.New
	mcpFactory.New = func(_ *vaultpkg.Vault, _ string, _ string) (*mcp.Server, error) {
		return nil, errors.New("agent not found")
	}
	t.Cleanup(func() { mcpFactory.New = origNew })

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	vaultDir, _ := vaultPath()
	tokenBytes, err := os.ReadFile(filepath.Join(vaultDir, "mcp-token"))
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	}
	payload, _ := json.Marshal(msg)

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-OpenPass-Agent", "test-agent")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("handler creation error status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}

	var errResp mcp.Message
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if errResp.Error == nil {
		t.Fatal("expected error in response")
	}
	if errResp.Error.Code != mcp.ErrCodeInternalError {
		t.Errorf("error code = %d, want %d", errResp.Error.Code, mcp.ErrCodeInternalError)
	}
	if !strings.Contains(errResp.Error.Message, "agent not found") {
		t.Errorf("error message = %q, want contains 'agent not found'", errResp.Error.Message)
	}
}

func TestRunHTTPServer_HandlerCacheHit(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	var callCount int
	origNew := mcpFactory.New
	mcpFactory.New = func(vault *vaultpkg.Vault, agentName string, transport string) (*mcp.Server, error) {
		callCount++
		return origNew(vault, agentName, transport)
	}
	t.Cleanup(func() { mcpFactory.New = origNew })

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	vaultDir, _ := vaultPath()
	tokenBytes, err := os.ReadFile(filepath.Join(vaultDir, "mcp-token"))
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
		},
	}
	payload, _ := json.Marshal(msg)

	req1, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewReader(payload))
	req1.Header.Set("Authorization", "Bearer "+token)
	req1.Header.Set("X-OpenPass-Agent", "default")
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Accept", "application/json, text/event-stream")

	resp1, err := newTestHTTPClient().Do(req1)
	if err != nil {
		t.Fatalf("first mcp request failed: %v", err)
	}
	_ = resp1.Body.Close()

	req2, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewReader(payload))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("X-OpenPass-Agent", "default")
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept", "application/json, text/event-stream")

	resp2, err := newTestHTTPClient().Do(req2)
	if err != nil {
		t.Fatalf("second mcp request failed: %v", err)
	}
	_ = resp2.Body.Close()

	if callCount != 1 {
		t.Errorf("mcpFactory.New called %d times, want 1 (cache hit expected)", callCount)
	}
}

func TestRunHTTPServer_CustomTokenPath(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	customTokenPath := filepath.Join(t.TempDir(), "custom-token")
	tokenContent := "custom-test-token-12345"
	if err := os.WriteFile(customTokenPath, []byte(tokenContent+"\n"), 0o600); err != nil {
		t.Fatalf("write custom token: %v", err)
	}

	v.Config.MCP = &config.MCPConfig{
		HTTPTokenFile: customTokenPath,
	}

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+tokenContent)
	req.Header.Set("X-OpenPass-Agent", "default")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("custom token auth failed: status = %d, want not %d", resp.StatusCode, http.StatusUnauthorized)
	}

	req2, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader("{}"))
	req2.Header.Set("Authorization", "Bearer wrong-token")
	req2.Header.Set("X-OpenPass-Agent", "default")
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept", "application/json, text/event-stream")

	resp2, err := newTestHTTPClient().Do(req2)
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	_ = resp2.Body.Close()

	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong token status = %d, want %d", resp2.StatusCode, http.StatusUnauthorized)
	}
}

func TestRunHTTPServer_HandleMessageError(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsync(ctx, t, port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	vaultDir, _ := vaultPath()
	tokenBytes, err := os.ReadFile(filepath.Join(vaultDir, "mcp-token"))
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_entries",
			"arguments": map[string]any{},
		},
	}
	payload, _ := json.Marshal(msg)

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-OpenPass-Agent", "default")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("mcp request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("tools/call status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var errResp mcp.Message
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if errResp.Error == nil {
		t.Fatal("expected error in response for uninitialized tools/call")
	}
	if errResp.Error.Code != mcp.ErrCodeServerError {
		t.Errorf("error code = %d, want %d", errResp.Error.Code, mcp.ErrCodeServerError)
	}
}

func TestServeCommand_StdioOnlyDoesNotStartHTTP(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	var stdioStarted bool
	var httpStarted bool
	runStdioServerFunc = func(_ context.Context, _ *vaultpkg.Vault, agentName string) error {
		stdioStarted = true
		if agentName != "default" {
			t.Errorf("agentName = %q, want default", agentName)
		}
		return nil
	}
	runHTTPServerFunc = func(_ context.Context, _ string, _ int, _ *vaultpkg.Vault) error {
		httpStarted = true
		return nil
	}
	serveSignalNotify = func(_ chan<- os.Signal, _ ...os.Signal) {}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "serve", "--stdio", "--agent", "default"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("serve command failed: %v", err)
	}
	if !stdioStarted {
		t.Fatal("stdio server was not started")
	}
	if httpStarted {
		t.Fatal("http server must not start in stdio-only mode")
	}
}

func TestServeCommand_ActiveSessionUsesNonInteractiveUnlock(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	sessionIsExpired = func(string) bool { return false }
	defer func() { sessionIsExpired = session.IsSessionExpired }()

	var unlockCalls []bool
	serveUnlockVault = func(_ string, interactive bool) (*vaultpkg.Vault, error) {
		unlockCalls = append(unlockCalls, interactive)
		return &vaultpkg.Vault{}, nil
	}

	runHTTPServerFunc = func(_ context.Context, _ string, _ int, _ *vaultpkg.Vault) error {
		return nil
	}
	serveSignalNotify = func(_ chan<- os.Signal, _ ...os.Signal) {}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "serve", "--port", "18080"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("serve command failed: %v", err)
	}

	if len(unlockCalls) != 1 {
		t.Fatalf("expected 1 unlock call, got %d: %v", len(unlockCalls), unlockCalls)
	}
	if unlockCalls[0] != false {
		t.Errorf("expected non-interactive unlock (interactive=false) for active session, got interactive=%v", unlockCalls[0])
	}
}

func TestServeCommand_ExpiredSessionUsesInteractiveUnlock(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	sessionIsExpired = func(string) bool { return true }
	defer func() { sessionIsExpired = session.IsSessionExpired }()

	var unlockCalls []bool
	serveUnlockVault = func(_ string, interactive bool) (*vaultpkg.Vault, error) {
		unlockCalls = append(unlockCalls, interactive)
		return nil, nil
	}

	runHTTPServerFunc = func(_ context.Context, _ string, _ int, _ *vaultpkg.Vault) error {
		return nil
	}
	serveSignalNotify = func(_ chan<- os.Signal, _ ...os.Signal) {}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "serve", "--port", "18081"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("serve command failed: %v", err)
	}

	if len(unlockCalls) != 1 {
		t.Fatalf("expected 1 unlock call, got %d: %v", len(unlockCalls), unlockCalls)
	}
	if unlockCalls[0] != true {
		t.Errorf("expected interactive unlock (interactive=true) for expired session, got interactive=%v", unlockCalls[0])
	}
}

func TestServeCommand_ActiveSessionFallbackToInteractive(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	sessionIsExpired = func(string) bool { return false }
	defer func() { sessionIsExpired = session.IsSessionExpired }()

	var unlockCalls []bool
	serveUnlockVault = func(_ string, interactive bool) (*vaultpkg.Vault, error) {
		unlockCalls = append(unlockCalls, interactive)
		if !interactive {
			return nil, fmt.Errorf("non-interactive unlock failed")
		}
		return nil, nil
	}

	runHTTPServerFunc = func(_ context.Context, _ string, _ int, _ *vaultpkg.Vault) error {
		return nil
	}
	serveSignalNotify = func(_ chan<- os.Signal, _ ...os.Signal) {}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "serve", "--port", "18082"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("serve command failed: %v", err)
	}

	if len(unlockCalls) != 2 {
		t.Fatalf("expected 2 unlock calls (non-interactive + fallback), got %d: %v", len(unlockCalls), unlockCalls)
	}
	if unlockCalls[0] != false {
		t.Errorf("expected first call to be non-interactive, got interactive=%v", unlockCalls[0])
	}
	if unlockCalls[1] != true {
		t.Errorf("expected second call to be interactive fallback, got interactive=%v", unlockCalls[1])
	}
}

func TestRunStdioServer_WithNilVault(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Replace os.Stdin / os.Stdout so stdio transport doesn't block on real TTY
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdin = r
	pr, pw, _ := os.Pipe()
	os.Stdout = pw

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Cancel immediately so transport.Start returns quickly
		cancel()
		_ = runStdioServer(ctx, nil, "")
	}()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("runStdioServer did not return in time")
	}

	wg.Wait()

	os.Stdin = oldStdin
	os.Stdout = oldStdout
	_ = r.Close()
	_ = w.Close()
	_ = pr.Close()
	_ = pw.Close()
}

func TestRunHTTPServer_MetricsEndpoint_NonLoopback_RequiresAuth(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsyncWithBind(ctx, t, "0.0.0.0", port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Request without auth should be denied
	resp, err := newTestHTTPClient().Get(baseURL + "/metrics")
	if err != nil {
		t.Fatalf("metrics request failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("metrics without auth status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Request with valid bearer token should succeed
	token := testMCPToken(t)
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = newTestHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("metrics request with auth failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("metrics with auth status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestRunHTTPServer_MetricsEndpoint_NonLoopback_AllowsWhenDisabled(t *testing.T) {
	v := newTestVault(t)
	v.Config.MCP = &config.MCPConfig{
		MetricsAuthRequired: false,
	}
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsyncWithBind(ctx, t, "0.0.0.0", port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	resp, err := newTestHTTPClient().Get(baseURL + "/metrics")
	if err != nil {
		t.Fatalf("metrics request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("metrics status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestRunHTTPServer_HealthEndpoint_NonLoopback(t *testing.T) {
	v := newTestVault(t)
	port := findFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	waitForServer := runHTTPServerAsyncWithBind(ctx, t, "0.0.0.0", port, v)
	defer func() {
		cancel()
		waitForServer()
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	resp, err := newTestHTTPClient().Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
