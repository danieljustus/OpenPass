package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/vault"
)

func TestHandleGet_Success(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	result, err := srv.handleGet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGet() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleGet() returned error: %s", result.Text)
	}

	var entry vault.Entry
	if err := json.Unmarshal([]byte(result.Text), &entry); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if entry.Data["password"] != "testpass123" {
		t.Errorf("password = %v, want testpass123", entry.Data["password"])
	}
}

func TestHandleGet_OutsideScope(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"work/"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	_, err := srv.handleGet(context.Background(), req)
	if err == nil {
		t.Fatal("handleGet() expected error for out-of-scope path, got nil")
	}
	if !strings.Contains(err.Error(), "outside allowed scope") {
		t.Fatalf("handleGet() error = %v, want 'outside allowed scope'", err)
	}
}

func TestHandleGet_MissingPath(t *testing.T) {
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

	result, err := srv.handleGet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGet() returned nil result")
	}
	if !result.IsError {
		t.Error("handleGet() expected error result for missing path")
	}
}

func TestHandleGet_NotFound(t *testing.T) {
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

	result, err := srv.handleGet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGet() error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("handleGet() expected error result for nonexistent entry")
	}
}

func TestHandleGet_WithMetadata(t *testing.T) {
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
			"path":             "github",
			"include_metadata": "true",
		},
	}

	result, err := srv.handleGet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGet() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleGet() returned error: %s", result.Text)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(result.Text), &response); err != nil {
		t.Fatalf("parse result: %v", err)
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' field in response")
	}
	if data["password"] != "testpass123" {
		t.Errorf("password = %v, want testpass123", data["password"])
	}

	meta, ok := response["meta"].(map[string]any)
	if !ok {
		t.Fatal("expected 'meta' field in response")
	}
	if meta["version"] != float64(1) {
		t.Errorf("version = %v, want 1", meta["version"])
	}
	if meta["created"] == nil || meta["created"] == "" {
		t.Error("created timestamp should be set")
	}
	if meta["updated"] == nil || meta["updated"] == "" {
		t.Error("updated timestamp should be set")
	}
}

func TestHandleGet_WithoutMetadata(t *testing.T) {
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
			"path": "github",
		},
	}

	result, err := srv.handleGet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGet() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleGet() returned error: %s", result.Text)
	}

	// Standard format: Entry struct with Data and Metadata fields
	var entry vault.Entry
	if err := json.Unmarshal([]byte(result.Text), &entry); err != nil {
		t.Fatalf("parse result: %v", err)
	}

	// Verify the entry data is accessible
	if entry.Data["password"] != "testpass123" {
		t.Errorf("password = %v, want testpass123", entry.Data["password"])
	}

	// Standard format includes metadata
	if entry.Metadata.Version != 1 {
		t.Errorf("version = %d, want 1", entry.Metadata.Version)
	}
}

func TestHandleGet_RedactedTOTPStillGeneratesCode(t *testing.T) {
	vaultDir := t.TempDir()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}

	cfg := &config.Config{
		DefaultAgent: "test",
		Agents: map[string]config.AgentProfile{
			"restricted": {
				Name:         "restricted",
				AllowedPaths: []string{"*"},
				CanWrite:     false,
				ApprovalMode: "none",
				RedactFields: []string{"totp.secret"},
			},
		},
	}
	if initErr := vault.Init(vaultDir, identity, cfg); initErr != nil {
		t.Fatalf("vault.Init() error = %v", initErr)
	}

	entry := &vault.Entry{
		Data: map[string]any{
			"password": "testpass",
			"totp": map[string]any{
				"secret":    "JBSWY3DPEHPK3PXP",
				"algorithm": "SHA1",
				"digits":    float64(6),
				"period":    float64(30),
			},
		},
	}
	if writeErr := vault.WriteEntry(vaultDir, "github", entry, identity); writeErr != nil {
		t.Fatalf("WriteEntry() error = %v", writeErr)
	}

	srv := &Server{
		vault: &vault.Vault{
			Dir:      vaultDir,
			Identity: identity,
		},
		agent: &config.AgentProfile{
			Name:         "restricted",
			AllowedPaths: []string{"*"},
			CanWrite:     false,
			ApprovalMode: "none",
			RedactFields: []string{"totp.secret"},
		},
	}

	getReq := CallToolRequest{
		Arguments: map[string]any{
			"path": "github",
		},
	}
	getResult, err := srv.handleGet(context.Background(), getReq)
	if err != nil {
		t.Fatalf("handleGet() error = %v", err)
	}

	var gotEntry map[string]any
	if parseErr := json.Unmarshal([]byte(getResult.Text), &gotEntry); parseErr != nil {
		t.Fatalf("parse get result: %v", parseErr)
	}

	data, ok := gotEntry["data"].(map[string]any)
	if !ok {
		t.Fatal("data field missing or wrong type")
	}
	totp, ok := data["totp"].(map[string]any)
	if !ok {
		t.Fatal("totp field missing or wrong type")
	}
	if totp["secret"] != "[REDACTED]" {
		t.Errorf("totp.secret = %v, want [REDACTED]", totp["secret"])
	}

	totpReq := CallToolRequest{
		Arguments: map[string]any{
			"path": "github",
		},
	}
	totpResult, err := srv.handleGenerateTOTP(context.Background(), totpReq)
	if err != nil {
		t.Fatalf("handleGenerateTOTP() error = %v", err)
	}

	var codeResult map[string]any
	if err := json.Unmarshal([]byte(totpResult.Text), &codeResult); err != nil {
		t.Fatalf("parse totp result: %v", err)
	}
	if codeResult["code"] == nil || codeResult["code"] == "" {
		t.Error("generate_totp returned empty code")
	}
}

func TestExecuteTool_GetEntry(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	args := json.RawMessage(`{"path": "github"}`)
	result, err := srv.executeTool(context.Background(), "get_entry", args)
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
}

func TestHandleGetMetadata_Success(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	result, err := srv.handleGetMetadata(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetMetadata() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGetMetadata() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleGetMetadata() returned error: %s", result.Text)
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(result.Text), &meta); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if meta["path"] != "github" {
		t.Errorf("path = %v, want github", meta["path"])
	}
	if meta["exists"] != true {
		t.Errorf("exists = %v, want true", meta["exists"])
	}
	if meta["version"] != float64(1) {
		t.Errorf("version = %v, want 1", meta["version"])
	}
	if meta["created"] == nil || meta["created"] == "" {
		t.Error("created timestamp should be set")
	}
	if meta["updated"] == nil || meta["updated"] == "" {
		t.Error("updated timestamp should be set")
	}
}

func TestHandleGetMetadata_OutsideScope(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"work/"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	_, err := srv.handleGetMetadata(context.Background(), req)
	if err == nil {
		t.Fatal("handleGetMetadata() expected error for out-of-scope path, got nil")
	}
	if !strings.Contains(err.Error(), "outside allowed scope") {
		t.Fatalf("handleGetMetadata() error = %v, want 'outside allowed scope'", err)
	}
}

func TestHandleGetMetadata_MissingPath(t *testing.T) {
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

	result, err := srv.handleGetMetadata(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetMetadata() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGetMetadata() returned nil result")
	}
	if !result.IsError {
		t.Error("handleGetMetadata() expected error result for missing path")
	}
}

func TestHandleGetMetadata_NotFound(t *testing.T) {
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

	result, err := srv.handleGetMetadata(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetMetadata() error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("handleGetMetadata() expected error result for nonexistent entry")
	}
}

func TestHandleGetMetadata_VersionIncrementedAfterUpdate(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	// Get initial metadata
	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}
	result, err := srv.handleGetMetadata(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetMetadata() initial error = %v", err)
	}

	var initialMeta map[string]any
	if unmarshalErr := json.Unmarshal([]byte(result.Text), &initialMeta); unmarshalErr != nil {
		t.Fatalf("parse initial result: %v", unmarshalErr)
	}
	initialVersion, _ := initialMeta["version"].(float64)

	// Update the entry
	setReq := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "password",
			"value": "StrongP@ssw0rd123",
		},
	}
	_, err = srv.handleSet(context.Background(), setReq)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}

	// Get metadata again
	result, err = srv.handleGetMetadata(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGetMetadata() after update error = %v", err)
	}

	var updatedMeta map[string]any
	if err := json.Unmarshal([]byte(result.Text), &updatedMeta); err != nil {
		t.Fatalf("parse updated result: %v", err)
	}
	updatedVersion, _ := updatedMeta["version"].(float64)

	if updatedVersion <= initialVersion {
		t.Errorf("version should increment after update: initial=%v, updated=%v", initialVersion, updatedVersion)
	}
}
