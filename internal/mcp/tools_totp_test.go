package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/vault"
)

func TestHandleGenerateTOTP_Success(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Create entry with TOTP configuration
	entry := &vault.Entry{
		Data: map[string]any{
			"password": "testpass123",
			"username": "testuser",
			"totp": map[string]any{
				"secret":       "JBSWY3DPEHPK3PXP",
				"issuer":       "GitHub",
				"account_name": "testuser",
			},
		},
	}
	if err := vault.WriteEntry(vaultDir, "github", entry, identity); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	result, err := srv.handleGenerateTOTP(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerateTOTP() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGenerateTOTP() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleGenerateTOTP() returned error: %s", result.Text)
	}

	// Verify result contains expected fields
	var totpResult map[string]any
	if err := json.Unmarshal([]byte(result.Text), &totpResult); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if totpResult["code"] == nil || totpResult["code"] == "" {
		t.Error("expected TOTP code in result")
	}
	if totpResult["expires_at"] == nil {
		t.Error("expected expires_at in result")
	}
	if totpResult["period"] == nil {
		t.Error("expected period in result")
	}
}

func TestHandleGenerateTOTP_OutsideScope(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"work/"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Create entry with TOTP configuration
	entry := &vault.Entry{
		Data: map[string]any{
			"password": "testpass123",
			"totp": map[string]any{
				"secret": "JBSWY3DPEHPK3PXP",
			},
		},
	}
	if err := vault.WriteEntry(vaultDir, "github", entry, identity); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	_, err := srv.handleGenerateTOTP(context.Background(), req)
	if err == nil {
		t.Fatal("handleGenerateTOTP() expected error for out-of-scope path, got nil")
	}
	if !strings.Contains(err.Error(), "outside allowed scope") {
		t.Fatalf("handleGenerateTOTP() error = %v, want 'outside allowed scope'", err)
	}
}

func TestHandleGenerateTOTP_MissingPath(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{},
	}

	result, err := srv.handleGenerateTOTP(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerateTOTP() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGenerateTOTP() returned nil result")
	}
	if !result.IsError {
		t.Error("handleGenerateTOTP() expected error result for missing path")
	}
}

func TestHandleGenerateTOTP_EntryNotFound(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "nonexistent"},
	}

	result, err := srv.handleGenerateTOTP(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerateTOTP() error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("handleGenerateTOTP() expected error result for nonexistent entry")
	}
}

func TestHandleGenerateTOTP_NoTOTPConfig(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Create entry WITHOUT TOTP configuration
	entry := &vault.Entry{
		Data: map[string]any{
			"password": "testpass123",
		},
	}
	if err := vault.WriteEntry(vaultDir, "github", entry, identity); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	_, err := srv.handleGenerateTOTP(context.Background(), req)
	if err == nil {
		t.Fatal("handleGenerateTOTP() expected error for entry without TOTP, got nil")
	}
	if !strings.Contains(err.Error(), "does not have TOTP configuration") {
		t.Fatalf("handleGenerateTOTP() error = %v, want 'does not have TOTP configuration'", err)
	}
}

func TestHandleGenerateTOTP_EmptySecret(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Create entry with empty TOTP secret
	entry := &vault.Entry{
		Data: map[string]any{
			"password": "testpass123",
			"totp": map[string]any{
				"secret": "",
			},
		},
	}
	if err := vault.WriteEntry(vaultDir, "github", entry, identity); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	_, err := srv.handleGenerateTOTP(context.Background(), req)
	if err == nil {
		t.Fatal("handleGenerateTOTP() expected error for entry with empty TOTP secret, got nil")
	}
}

func TestExecuteTool_GenerateTOTP(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Create entry with TOTP configuration
	entry := &vault.Entry{
		Data: map[string]any{
			"password": "testpass123",
			"totp": map[string]any{
				"secret": "JBSWY3DPEHPK3PXP",
			},
		},
	}
	if err := vault.WriteEntry(vaultDir, "github", entry, identity); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	args := json.RawMessage(`{"path": "github"}`)
	result, err := srv.executeTool(context.Background(), "generate_totp", args)
	if err != nil {
		t.Fatalf("executeTool() error = %v", err)
	}

	content, ok := result["content"].([]map[string]any)
	if !ok {
		t.Fatal("result content has unexpected type")
	}
	if len(content) == 0 {
		t.Fatal("expected content in result")
	}
	if result["isError"] == true {
		t.Errorf("expected no error, got isError=true: %s", content[0]["text"])
	}
}
