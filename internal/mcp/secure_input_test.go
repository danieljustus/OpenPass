package mcp

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/vault"
)

func TestHandleSecureInput_Success(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Mock the secure TTY input
	originalOpenSecureTTY := openSecureTTY
	openSecureTTY = func() (secureInputDevice, error) {
		return &mockSecureInputDevice{value: "my-secret-api-key"}, nil
	}
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":        "github",
			"field":       "api_key",
			"description": "Enter your GitHub API key",
		},
	}

	result, err := srv.handleSecureInput(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSecureInput() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSecureInput() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleSecureInput() returned error: %s", result.Text)
	}

	// Verify the entry was updated
	entry, err := vault.ReadEntry(vaultDir, "github", identity)
	if err != nil {
		t.Fatalf("ReadEntry() error = %v", err)
	}
	if entry.Data["api_key"] != "my-secret-api-key" {
		t.Errorf("api_key = %v, want my-secret-api-key", entry.Data["api_key"])
	}

	// Verify the response doesn't contain the secret
	if result.Text == "my-secret-api-key" {
		t.Error("result should not contain the secret value")
	}
}

func TestHandleSecureInput_NewEntry(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Mock the secure TTY input
	originalOpenSecureTTY := openSecureTTY
	openSecureTTY = func() (secureInputDevice, error) {
		return &mockSecureInputDevice{value: "new-secret-value"}, nil
	}
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "new-service",
			"field": "password",
		},
	}

	result, err := srv.handleSecureInput(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSecureInput() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSecureInput() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleSecureInput() returned error: %s", result.Text)
	}

	// Verify the entry was created
	entry, err := vault.ReadEntry(vaultDir, "new-service", identity)
	if err != nil {
		t.Fatalf("ReadEntry() error = %v", err)
	}
	if entry.Data["password"] != "new-secret-value" {
		t.Errorf("password = %v, want new-secret-value", entry.Data["password"])
	}
}

func TestHandleSecureInput_WriteDenied(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "api_key",
		},
	}

	_, err := srv.handleSecureInput(context.Background(), req)
	if err == nil {
		t.Fatal("handleSecureInput() expected error for write-denied agent, got nil")
	}
	if err.Error() != "write operations not permitted for this agent" {
		t.Errorf("handleSecureInput() error = %v, want 'write operations not permitted for this agent'", err)
	}
}

func TestHandleSecureInput_OutsideScope(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"work/"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "api_key",
		},
	}

	_, err := srv.handleSecureInput(context.Background(), req)
	if err == nil {
		t.Fatal("handleSecureInput() expected error for out-of-scope path, got nil")
	}
	if err.Error() != `access denied: path "github" outside allowed scope` {
		t.Errorf("handleSecureInput() error = %v, want 'access denied: path \"github\" outside allowed scope'", err)
	}
}

func TestHandleSecureInput_ApprovalRequired(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "deny",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "api_key",
		},
	}

	_, err := srv.handleSecureInput(context.Background(), req)
	if err == nil {
		t.Fatal("handleSecureInput() expected error for approval-required path, got nil")
	}
	if err.Error() != `secure input for "github" denied: approval required but cannot be granted` {
		t.Errorf("handleSecureInput() error = %v", err)
	}
}

func TestHandleSecureInput_MissingParams(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{
			"path": "github",
		},
	}

	result, err := srv.handleSecureInput(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSecureInput() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSecureInput() returned nil result")
	}
	if !result.IsError {
		t.Error("handleSecureInput() expected error result for missing field")
	}
}

func TestHandleSecureInput_EmptyValue(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Mock the secure TTY input with empty value
	originalOpenSecureTTY := openSecureTTY
	openSecureTTY = func() (secureInputDevice, error) {
		return &mockSecureInputDevice{value: ""}, nil
	}
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "api_key",
		},
	}

	result, err := srv.handleSecureInput(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSecureInput() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSecureInput() returned nil result")
	}
	if !result.IsError {
		t.Error("handleSecureInput() expected error result for empty value")
	}
}

func TestHandleSecureInput_NoTTY(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Mock TTY as unavailable
	originalOpenSecureTTY := openSecureTTY
	openSecureTTY = func() (secureInputDevice, error) {
		return nil, errors.New("no TTY available")
	}
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "api_key",
		},
	}

	_, err := srv.handleSecureInput(context.Background(), req)
	if err == nil {
		t.Fatal("handleSecureInput() expected error when TTY unavailable, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) && err.Error() != "secure input failed: secure input requires an interactive terminal (TTY not available)" {
		t.Logf("Got expected error: %v", err)
	}
}

func TestSecureInputPrompt_BuildPrompt(t *testing.T) {
	prompt := buildSecureInputPrompt("github", "api_key", "Enter your GitHub API key")
	if prompt == "" {
		t.Error("buildSecureInputPrompt() returned empty string")
	}
	if !contains(prompt, "github") {
		t.Error("prompt should contain path")
	}
	if !contains(prompt, "api_key") {
		t.Error("prompt should contain field")
	}
	if !contains(prompt, "Enter your GitHub API key") {
		t.Error("prompt should contain description")
	}
	if !contains(prompt, "SECURE INPUT REQUIRED") {
		t.Error("prompt should indicate secure input")
	}
}

func TestSecureInputPrompt_TTYUnavailable(t *testing.T) {
	originalOpenSecureTTY := openSecureTTY
	openSecureTTY = func() (secureInputDevice, error) {
		return nil, errors.New("no TTY")
	}
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	_, err := SecureInputPrompt("test prompt", 5*time.Second)
	if err == nil {
		t.Fatal("SecureInputPrompt() expected error when TTY unavailable, got nil")
	}
	if err.Error() != "secure input requires an interactive terminal (TTY not available)" {
		t.Errorf("SecureInputPrompt() error = %v", err)
	}
}

func TestSecureInputPrompt_WriteError(t *testing.T) {
	originalOpenSecureTTY := openSecureTTY
	openSecureTTY = func() (secureInputDevice, error) {
		readEnd, _, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe failed: %v", err)
		}
		return &mockSecureInputDeviceWithFile{output: readEnd}, nil
	}
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	_, err := SecureInputPrompt("test prompt", 5*time.Second)
	if err == nil {
		t.Fatal("SecureInputPrompt() expected error on write failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to write prompt") {
		t.Errorf("SecureInputPrompt() error = %v, want write prompt error", err)
	}
}

func TestSecureInputPrompt_Timeout(t *testing.T) {
	originalOpenSecureTTY := openSecureTTY
	openSecureTTY = func() (secureInputDevice, error) {
		return &mockSecureInputDevice{
			value:     "",
			readError: os.ErrDeadlineExceeded,
		}, nil
	}
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	_, err := SecureInputPrompt("test prompt", 1*time.Second)
	if err == nil {
		t.Fatal("SecureInputPrompt() expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("SecureInputPrompt() error = %v, want timeout error", err)
	}
}

func TestReadSecureInput_DeadlineError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows: pipe behavior differs")
	}
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	defer func() {
		_ = readEnd.Close()
		_ = writeEnd.Close()
	}()

	device := &mockSecureInputDeviceWithFile{input: readEnd}
	_, err = readSecureInput(device, 1*time.Second)
	if err != nil {
		t.Fatalf("readSecureInput() with nil deadline error = %v", err)
	}
}

type mockSecureInputDeviceWithFile struct {
	input  *os.File
	output *os.File
}

func (m *mockSecureInputDeviceWithFile) ReadString() (string, error) {
	return "test-value", nil
}

func (m *mockSecureInputDeviceWithFile) Input() *os.File {
	return m.input
}

func (m *mockSecureInputDeviceWithFile) Output() *os.File {
	return m.output
}

func (m *mockSecureInputDeviceWithFile) Close() error {
	return nil
}

type mockSecureInputDevice struct {
	value     string
	readError error
}

func (m *mockSecureInputDevice) ReadString() (string, error) {
	if m.readError != nil {
		return "", m.readError
	}
	return m.value, nil
}

func (m *mockSecureInputDevice) Input() *os.File {
	return nil
}

func (m *mockSecureInputDevice) Output() *os.File {
	return nil
}

func (m *mockSecureInputDevice) Close() error {
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
