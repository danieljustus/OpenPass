package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/danieljustus/OpenPass/internal/audit"
	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/vault"
)

//nolint:unparam // transport always "stdio" in current test suite
func newTestServerWithVault(t *testing.T, profile config.AgentProfile, transport string, vaultDir string) *Server {
	t.Helper()

	auditLog, err := audit.New("test")
	if err != nil {
		t.Fatalf("audit.New() error = %v", err)
	}

	var identity *age.X25519Identity
	if vaultDir != "" {
		identity, err = age.GenerateX25519Identity()
		if err != nil {
			t.Fatalf("generate identity: %v", err)
		}
	}

	return &Server{
		vault: &vault.Vault{
			Dir:      vaultDir,
			Identity: identity,
		},
		agent:     &profile,
		auditLog:  auditLog,
		transport: transport,
	}
}

// mockVault creates a temp vault directory with entries for testing
func mockVault(t *testing.T) (string, *age.X25519Identity) {
	t.Helper()

	dir := t.TempDir()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}

	// Create an entry
	entry := &vault.Entry{
		Data: map[string]any{
			"password": "testpass123",
			"username": "testuser",
		},
	}
	if err := vault.WriteEntry(dir, "github", entry, identity); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	return dir, identity
}

func TestHandleList_WithPrefix(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"prefix": ""},
	}

	result, err := srv.handleList(context.Background(), req)
	if err != nil {
		t.Fatalf("handleList() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleList() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleList() returned error: %s", result.Text)
	}

	var entries []string
	if err := json.Unmarshal([]byte(result.Text), &entries); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries, got empty list")
	}
}

func TestHandleList_OutsideScope(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"work/"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"prefix": ""},
	}

	_, err := srv.handleList(context.Background(), req)
	if err == nil {
		t.Fatal("handleList() expected error for out-of-scope path, got nil")
	}
	if !strings.Contains(err.Error(), "outside allowed scope") {
		t.Fatalf("handleList() error = %v, want 'outside allowed scope'", err)
	}
}

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

	_, err := srv.handleGet(context.Background(), req)
	if err == nil {
		t.Fatal("handleGet() expected error for nonexistent entry, got nil")
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

	_, err := srv.handleGetMetadata(context.Background(), req)
	if err == nil {
		t.Fatal("handleGetMetadata() expected error for nonexistent entry, got nil")
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

func TestHandleFind_Success(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"query": "test"},
	}

	result, err := srv.handleFind(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFind() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleFind() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleFind() returned error: %s", result.Text)
	}

	var matches []vault.Match
	if err := json.Unmarshal([]byte(result.Text), &matches); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	// github entry should match since it contains "test" in username
	found := false
	for _, m := range matches {
		if m.Path == "github" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find github entry in matches")
	}
}

func TestHandleFind_MissingQuery(t *testing.T) {
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

	result, err := srv.handleFind(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFind() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleFind() returned nil result")
	}
	if !result.IsError {
		t.Error("handleFind() expected error result for missing query")
	}
}

func TestHandleFind_FiltersByScope(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"work/"}, // Only allow work/ paths
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"query": "test"},
	}

	result, err := srv.handleFind(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFind() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleFind() returned nil result")
	}

	var matches []vault.Match
	if err := json.Unmarshal([]byte(result.Text), &matches); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	// github is not in work/ scope, so should not be in results
	for _, m := range matches {
		if m.Path == "github" {
			t.Error("github should not be in results due to scope filtering")
		}
	}
}

func TestHandleFind_DoesNotDecryptOutOfScopeEntries(t *testing.T) {
	vaultDir, identity := mockVault(t)
	if err := vault.WriteEntry(vaultDir, "work/allowed", &vault.Entry{
		Data: map[string]any{"password": "allowed-secret"},
	}, identity); err != nil {
		t.Fatalf("write allowed entry: %v", err)
	}

	privateDir := filepath.Join(vaultDir, "entries", "private")
	if err := os.MkdirAll(privateDir, 0o700); err != nil {
		t.Fatalf("create private dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(privateDir, "broken.age"), []byte("not an age payload"), 0o600); err != nil {
		t.Fatalf("write corrupt out-of-scope entry: %v", err)
	}

	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"work/"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"query": "allowed"},
	}

	result, err := srv.handleFind(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFind() error = %v", err)
	}

	var matches []vault.Match
	if err := json.Unmarshal([]byte(result.Text), &matches); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if len(matches) != 1 || matches[0].Path != "work/allowed" {
		t.Fatalf("matches = %#v, want only work/allowed", matches)
	}
}

func TestHandleSet_WriteDenied(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false, // Cannot write
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "password",
			"value": "newpass",
		},
	}

	_, err := srv.handleSet(context.Background(), req)
	if err == nil {
		t.Fatal("handleSet() expected error for write-denied agent, got nil")
	}
	if !strings.Contains(err.Error(), "write operations not permitted") {
		t.Fatalf("handleSet() error = %v, want 'write operations not permitted'", err)
	}
}

func TestHandleSet_Success(t *testing.T) {
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
			"path":  "github",
			"field": "password",
			"value": "StrongP@ssw0rd123",
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleSet() returned error: %s", result.Text)
	}

	// Verify the entry was updated
	entry, err := vault.ReadEntry(vaultDir, "github", identity)
	if err != nil {
		t.Fatalf("ReadEntry() error = %v", err)
	}
	if entry.Data["password"] != "StrongP@ssw0rd123" {
		t.Errorf("password = %v, want StrongP@ssw0rd123", entry.Data["password"])
	}
}

func TestHandleSet_OutsideScope(t *testing.T) {
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
			"field": "password",
			"value": "newpass",
		},
	}

	_, err := srv.handleSet(context.Background(), req)
	if err == nil {
		t.Fatal("handleSet() expected error for out-of-scope path, got nil")
	}
	if !strings.Contains(err.Error(), "outside allowed scope") {
		t.Fatalf("handleSet() error = %v, want 'outside allowed scope'", err)
	}
}

func TestHandleSet_ApprovalRequired(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "deny", // Requires approval
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "password",
			"value": "newpass",
		},
	}

	_, err := srv.handleSet(context.Background(), req)
	if err == nil {
		t.Fatal("handleSet() expected error for approval-required path, got nil")
	}
	if !strings.Contains(err.Error(), "approval required") {
		t.Fatalf("handleSet() error = %v, want 'approval required'", err)
	}
}

func TestHandleSet_MissingParams(t *testing.T) {
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
			// missing field and value
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if !result.IsError {
		t.Error("handleSet() expected error result for missing params")
	}
}

func TestHandleSet_NewEntry(t *testing.T) {
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
			"path":  "newentry",
			"field": "password",
			"value": "StrongP@ssw0rd123",
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleSet() returned error: %s", result.Text)
	}

	// Verify the entry was created
	_, err = vault.ReadEntry(vaultDir, "newentry", identity)
	if err != nil {
		t.Fatalf("ReadEntry() error = %v", err)
	}
}

func assertTOTPSet(t *testing.T, vaultDir string, identity *age.X25519Identity, path string) {
	t.Helper()

	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	totpJSON := `{"secret":"JBSWY3DPEHPK3PXP","issuer":"GitHub","account_name":"testuser"}`
	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  path,
			"field": "totp",
			"value": totpJSON,
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleSet() returned error: %s", result.Text)
	}

	entry, err := vault.ReadEntry(vaultDir, path, identity)
	if err != nil {
		t.Fatalf("ReadEntry() error = %v", err)
	}
	totpData, ok := entry.Data["totp"].(map[string]any)
	if !ok {
		t.Fatal("totp field should be map[string]any")
	}
	if totpData["secret"] != "JBSWY3DPEHPK3PXP" {
		t.Errorf("totp.secret = %v, want JBSWY3DPEHPK3PXP", totpData["secret"])
	}
	if totpData["issuer"] != "GitHub" {
		t.Errorf("totp.issuer = %v, want GitHub", totpData["issuer"])
	}
	if totpData["account_name"] != "testuser" {
		t.Errorf("totp.account_name = %v, want testuser", totpData["account_name"])
	}
}

func TestHandleSet_TOTPField(t *testing.T) {
	vaultDir, identity := mockVault(t)
	assertTOTPSet(t, vaultDir, identity, "github")
}

func TestHandleSet_NewEntryTOTP(t *testing.T) {
	vaultDir, identity := mockVault(t)
	assertTOTPSet(t, vaultDir, identity, "newentry")
}

func TestHandleSet_InvalidTOTPJSON(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "totp",
			"value": "not-valid-json",
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if !result.IsError {
		t.Error("handleSet() expected error for invalid TOTP JSON")
	}
}

func TestHandleSet_TOTPInvalidAlgorithm(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	totpJSON := `{"secret":"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ","algorithm":"MD5","digits":6,"period":30}`
	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "totp",
			"value": totpJSON,
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if !result.IsError {
		t.Error("handleSet() expected error for invalid TOTP algorithm")
	}
	if !strings.Contains(result.Text, "invalid TOTP") {
		t.Errorf("handleSet() error text = %q, want to contain 'invalid TOTP'", result.Text)
	}
}

func TestHandleSet_TOTPInvalidDigits(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	totpJSON := `{"secret":"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ","algorithm":"SHA1","digits":7,"period":30}`
	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "totp",
			"value": totpJSON,
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if !result.IsError {
		t.Error("handleSet() expected error for invalid TOTP digits")
	}
	if !strings.Contains(result.Text, "invalid TOTP") {
		t.Errorf("handleSet() error text = %q, want to contain 'invalid TOTP'", result.Text)
	}
}

func TestHandleSet_TOTPInvalidPeriod(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	totpJSON := `{"secret":"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ","algorithm":"SHA1","digits":6,"period":5000}`
	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "github",
			"field": "totp",
			"value": totpJSON,
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if !result.IsError {
		t.Error("handleSet() expected error for invalid TOTP period")
	}
	if !strings.Contains(result.Text, "invalid TOTP") {
		t.Errorf("handleSet() error text = %q, want to contain 'invalid TOTP'", result.Text)
	}
}

func TestHandleSet_TOTPValidParams(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	totpJSON := `{"secret":"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ","algorithm":"SHA1","digits":6,"period":30}`
	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "newentry",
			"field": "totp",
			"value": totpJSON,
		},
	}

	result, err := srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleSet() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleSet() returned error: %s", result.Text)
	}

	entry, err := vault.ReadEntry(vaultDir, "newentry", identity)
	if err != nil {
		t.Fatalf("ReadEntry() error = %v", err)
	}
	totpData, ok := entry.Data["totp"].(map[string]any)
	if !ok {
		t.Fatal("totp field should be map[string]any")
	}
	if totpData["algorithm"] != "SHA1" {
		t.Errorf("totp.algorithm = %v, want SHA1", totpData["algorithm"])
	}
	if totpData["digits"] != float64(6) {
		t.Errorf("totp.digits = %v, want 6", totpData["digits"])
	}
	if totpData["period"] != float64(30) {
		t.Errorf("totp.period = %v, want 30", totpData["period"])
	}
}

func TestHandleGenerate_DefaultLength(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", "")

	req := CallToolRequest{
		Arguments: map[string]any{},
	}

	result, err := srv.handleGenerate(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerate() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGenerate() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleGenerate() returned error: %s", result.Text)
	}

	password := result.Text
	if len(password) != 16 {
		t.Errorf("password length = %d, want 16 (default)", len(password))
	}
}

func TestHandleGenerate_CustomLength(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", "")

	req := CallToolRequest{
		Arguments: map[string]any{"length": 32.0},
	}

	result, err := srv.handleGenerate(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerate() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGenerate() returned nil result")
	}

	password := result.Text
	if len(password) != 32 {
		t.Errorf("password length = %d, want 32", len(password))
	}
}

func TestHandleGenerate_WithSymbols(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", "")

	req := CallToolRequest{
		Arguments: map[string]any{"length": 50.0, "symbols": "true"},
	}

	result, err := srv.handleGenerate(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerate() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGenerate() returned nil result")
	}

	password := result.Text
	hasSymbol := false
	for _, c := range password {
		if strings.Contains("!@#$%^&*()_+-=[]{}|;:,.<>?", string(c)) {
			hasSymbol = true
			break
		}
	}
	if !hasSymbol {
		t.Error("expected password to contain symbols")
	}
}

func TestHandleGenerate_WithoutSymbols(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", "")

	req := CallToolRequest{
		Arguments: map[string]any{"length": 50.0, "symbols": "false"},
	}

	result, err := srv.handleGenerate(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerate() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGenerate() returned nil result")
	}

	password := result.Text
	for _, c := range password {
		if strings.Contains("!@#$%^&*()_+-=[]{}|;:,.<>?", string(c)) {
			t.Error("expected password to NOT contain symbols")
			break
		}
	}
}

func TestHandleDelete_WriteDenied(t *testing.T) {
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

	_, err := srv.handleDelete(context.Background(), req)
	if err == nil {
		t.Fatal("handleDelete() expected error for write-denied agent, got nil")
	}
	if !strings.Contains(err.Error(), "delete operations not permitted") {
		t.Fatalf("handleDelete() error = %v, want 'delete operations not permitted'", err)
	}
}

func TestHandleDelete_Success(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	result, err := srv.handleDelete(context.Background(), req)
	if err != nil {
		t.Fatalf("handleDelete() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleDelete() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleDelete() returned error: %s", result.Text)
	}

	// Verify the entry was deleted
	_, err = vault.ReadEntry(vaultDir, "github", identity)
	if err == nil {
		t.Error("expected entry to be deleted")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected IsNotExist error, got %v", err)
	}
}

func TestHandleDelete_OutsideScope(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"work/"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	_, err := srv.handleDelete(context.Background(), req)
	if err == nil {
		t.Fatal("handleDelete() expected error for out-of-scope path, got nil")
	}
	if !strings.Contains(err.Error(), "outside allowed scope") {
		t.Fatalf("handleDelete() error = %v, want 'outside allowed scope'", err)
	}
}

func TestHandleDelete_ApprovalRequired(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "deny",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "github"},
	}

	_, err := srv.handleDelete(context.Background(), req)
	if err == nil {
		t.Fatal("handleDelete() expected error for approval-required path, got nil")
	}
	if !strings.Contains(err.Error(), "approval required") {
		t.Fatalf("handleDelete() error = %v, want 'approval required'", err)
	}
}

func TestHandleDelete_NotFound(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{"path": "nonexistent"},
	}

	_, err := srv.handleDelete(context.Background(), req)
	if err == nil {
		t.Fatal("handleDelete() expected error for nonexistent entry, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("handleDelete() error = %v, want 'not found'", err)
	}
}

func TestHandleDelete_MissingPath(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	req := CallToolRequest{
		Arguments: map[string]any{},
	}

	result, err := srv.handleDelete(context.Background(), req)
	if err != nil {
		t.Fatalf("handleDelete() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleDelete() returned nil result")
	}
	if !result.IsError {
		t.Error("handleDelete() expected error result for missing path")
	}
}

func TestCollectFieldMatches(t *testing.T) {
	tests := []struct {
		data     map[string]any
		expected map[string]struct{}
		name     string
		needle   string
	}{
		{
			name: "simple match",
			data: map[string]any{
				"password": "secret123",
			},
			needle:   "secret",
			expected: map[string]struct{}{"password": {}},
		},
		{
			name: "nested match",
			data: map[string]any{
				"credentials": map[string]any{
					"api_key": "key123",
				},
			},
			needle:   "key",
			expected: map[string]struct{}{"credentials.api_key": {}},
		},
		{
			name: "no match",
			data: map[string]any{
				"password": "secret123",
			},
			needle:   "nomatch",
			expected: map[string]struct{}{},
		},
		{
			name:     "empty needle",
			data:     map[string]any{"password": "secret123"},
			needle:   "",
			expected: map[string]struct{}{"password": {}},
		},
		{
			name: "array match",
			data: map[string]any{
				"urls": []any{"https://example.com", "https://test.com"},
			},
			needle:   "example",
			expected: map[string]struct{}{"urls[0]": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]struct{})
			vault.CollectFieldMatches(result, "", tt.data, tt.needle)
			if len(result) != len(tt.expected) {
				t.Errorf("collectFieldMatches() got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	password, err := generatePassword(16, true)
	if err != nil {
		t.Fatalf("generatePassword() error = %v", err)
	}
	if len(password) != 16 {
		t.Errorf("password length = %d, want 16", len(password))
	}

	// Generate again to ensure randomness
	password2, err := generatePassword(16, true)
	if err != nil {
		t.Fatalf("generatePassword() second call error = %v", err)
	}
	if password == password2 {
		t.Error("expected different passwords on consecutive calls")
	}
}

func TestGeneratePassword_ZeroLength(t *testing.T) {
	password, err := generatePassword(0, true)
	if err != nil {
		t.Fatalf("generatePassword() error = %v", err)
	}
	// Should default to 16
	if len(password) != 16 {
		t.Errorf("password length = %d, want 16 (default)", len(password))
	}
}

func TestGeneratePassword_NoSymbols(t *testing.T) {
	password, err := generatePassword(50, false)
	if err != nil {
		t.Fatalf("generatePassword() error = %v", err)
	}

	for _, c := range password {
		if strings.Contains("!@#$%^&*()_+-=[]{}|;:,.<>?", string(c)) {
			t.Error("expected password without symbols")
			break
		}
	}
}

func TestFindEntries(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	matches, err := srv.findEntries("test")
	if err != nil {
		t.Fatalf("findEntries() error = %v", err)
	}

	if len(matches) == 0 {
		t.Error("expected matches for 'test' query")
	}
}

func TestFindEntries_NoResults(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	matches, err := srv.findEntries("zzzzz_nomatch")
	if err != nil {
		t.Fatalf("findEntries() error = %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("expected no matches, got %d", len(matches))
	}
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", "")

	// Use empty JSON object instead of nil to avoid parse error
	args := json.RawMessage(`{}`)
	_, err := srv.executeTool(context.Background(), "unknown_tool", args)
	if err == nil {
		t.Fatal("executeTool() expected error for unknown tool, got nil")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("executeTool() error = %v, want 'unknown tool'", err)
	}
}

func TestExecuteTool_ListEntries(t *testing.T) {
	vaultDir, identity := mockVault(t)
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	args := json.RawMessage(`{"prefix": ""}`)
	result, err := srv.executeTool(context.Background(), "list_entries", args)
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
	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("content text has unexpected type")
	}
	if !strings.Contains(text, "github") {
		t.Errorf("expected 'github' in result, got %s", text)
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

func TestToolError(t *testing.T) {
	result := toolError("test error")
	if result == nil {
		t.Fatal("toolError() returned nil")
	}
	if !result.IsError {
		t.Error("toolError() expected IsError to be true")
	}
}

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

	_, err := srv.handleGenerateTOTP(context.Background(), req)
	if err == nil {
		t.Fatal("handleGenerateTOTP() expected error for nonexistent entry, got nil")
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

func TestRegisterTools(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", "")

	mcpSrv := NewMCPServer("test", "1.0.0")
	srv.RegisterTools(mcpSrv)

	if len(mcpSrv.tools) == 0 {
		t.Error("RegisterTools() should register tools")
	}
}

func TestToolsListPayloadMatchesAvailableRegistry(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "http", "")

	tools := toolsListPayload(srv)
	names := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		names[name] = tool
	}

	for _, def := range availableToolDefinitions(srv) {
		if _, ok := names[def.Name]; !ok {
			t.Fatalf("tools/list missing available tool %q", def.Name)
		}
	}

	for _, name := range []string{"list_entries", "get_entry", "get_entry_metadata", "generate_password", "health", "delete_entry", "openpass_delete"} {
		if _, ok := names[name]; !ok {
			t.Fatalf("tools/list missing expected tool %q", name)
		}
	}
	if _, ok := names["secure_input"]; ok {
		t.Fatal("secure_input should not be listed for non-stdio transports")
	}

	getEntrySchema, ok := names["get_entry"]["inputSchema"].(map[string]any)
	if !ok {
		t.Fatal("get_entry inputSchema has unexpected type")
	}
	properties, ok := getEntrySchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("get_entry properties have unexpected type")
	}
	if _, ok := properties["include_metadata"]; !ok {
		t.Fatal("get_entry schema missing include_metadata")
	}
}

func TestSecureInputToolAvailabilityInRegistry(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     true,
		ApprovalMode: "none",
	}, "stdio", "")

	originalOpenSecureTTY := openSecureTTY
	defer func() { openSecureTTY = originalOpenSecureTTY }()

	openSecureTTY = func() (secureInputDevice, error) {
		return nil, os.ErrNotExist
	}
	tools := toolsListPayload(srv)
	if toolNamesContain(tools, "secure_input") {
		t.Fatal("secure_input should be hidden when no TTY is available")
	}

	openSecureTTY = func() (secureInputDevice, error) {
		return &mockSecureInputDevice{}, nil
	}

	tools = toolsListPayload(srv)
	if !toolNamesContain(tools, "secure_input") {
		t.Fatal("secure_input should be listed when stdio and TTY are available")
	}
}

func toolNamesContain(tools []map[string]any, target string) bool {
	for _, tool := range tools {
		if name, _ := tool["name"].(string); name == target {
			return true
		}
	}
	return false
}

func TestFindEntries_ListFails(t *testing.T) {
	vaultDir := t.TempDir()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", vaultDir)
	srv.vault.Identity = identity

	srv.vault.Dir = filepath.Join(vaultDir, "nonexistent")

	_, err = srv.findEntries("test")
	if err == nil {
		t.Fatal("findEntries() expected error for nonexistent dir, got nil")
	}
}

func TestCollectFieldMatches_EdgeCases(t *testing.T) {
	tests := []struct {
		data     map[string]any
		expected map[string]struct{}
		name     string
		needle   string
	}{
		{
			name:     "empty map",
			data:     map[string]any{},
			needle:   "test",
			expected: map[string]struct{}{},
		},
		{
			name: "deeply nested arrays",
			data: map[string]any{
				"deep": []any{
					[]any{
						[]any{"value"},
					},
				},
			},
			needle:   "value",
			expected: map[string]struct{}{"deep[0][0][0]": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]struct{})
			vault.CollectFieldMatches(result, "", tt.data, tt.needle)
			if len(result) != len(tt.expected) {
				t.Errorf("collectFieldMatches() got %v, want %v", result, tt.expected)
			}
			for k := range tt.expected {
				if _, ok := result[k]; !ok {
					t.Errorf("missing key %q", k)
				}
			}
		})
	}

	t.Run("scalar with empty prefix", func(t *testing.T) {
		result := make(map[string]struct{})
		vault.CollectFieldMatches(result, "", "testvalue", "test")
		if len(result) != 0 {
			t.Errorf("collectFieldMatches() with scalar and empty prefix should return empty, got %v", result)
		}
	})
}

func TestHandleSet_WriteEntryFails(t *testing.T) {
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
			"path":  "new-entry-that-does-not-exist",
			"field": "password",
			"value": "StrongP@ssw0rd123",
		},
	}

	srv.vault.Dir = "/nonexistent/directory/that/cannot/be/created"

	_, err := srv.handleSet(context.Background(), req)
	if err == nil {
		t.Fatal("handleSet() expected error when WriteEntry fails, got nil")
	}
}

func TestHandleGenerate_NonNumericLength(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", "")

	req := CallToolRequest{
		Arguments: map[string]any{"length": "not-a-number"},
	}

	result, err := srv.handleGenerate(context.Background(), req)
	if err != nil {
		t.Fatalf("handleGenerate() error = %v", err)
	}
	if result == nil {
		t.Fatal("handleGenerate() returned nil result")
	}
	if result.IsError {
		t.Fatalf("handleGenerate() returned error: %s", result.Text)
	}

	password := result.Text
	if len(password) != 16 {
		t.Errorf("password length = %d, want 16 (default fallback)", len(password))
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

func TestExecuteTool_GeneratePassword(t *testing.T) {
	srv := newTestServerWithVault(t, config.AgentProfile{
		Name:         "test",
		AllowedPaths: []string{"*"},
		CanWrite:     false,
		ApprovalMode: "none",
	}, "stdio", "")

	args := json.RawMessage(`{}`)
	result, err := srv.executeTool(context.Background(), "generate_password", args)
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

func TestHandleSet_PreservesMultiRecipientAccess(t *testing.T) {
	writerDir := t.TempDir()
	writerIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate writer identity: %v", err)
	}

	cfg := &config.Config{
		DefaultAgent: "test",
		Agents: map[string]config.AgentProfile{
			"test": {
				Name:         "test",
				AllowedPaths: []string{"*"},
				CanWrite:     true,
				ApprovalMode: "none",
			},
		},
	}
	if initErr := vault.Init(writerDir, writerIdentity, cfg); initErr != nil {
		t.Fatalf("vault.Init() error = %v", initErr)
	}

	secondIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate second identity: %v", err)
	}

	rm := vault.NewRecipientsManager(writerDir)
	if addErr := rm.AddRecipient(secondIdentity.Recipient().String()); addErr != nil {
		t.Fatalf("AddRecipient() error = %v", addErr)
	}

	srv := &Server{
		vault: &vault.Vault{
			Dir:      writerDir,
			Identity: writerIdentity,
		},
		agent: &config.AgentProfile{
			Name:         "test",
			AllowedPaths: []string{"*"},
			CanWrite:     true,
			ApprovalMode: "none",
		},
	}

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "shared-entry",
			"field": "password",
			"value": "StrongP@ssw0rd123",
		},
	}
	_, err = srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}

	entry, err := vault.ReadEntry(writerDir, "shared-entry", secondIdentity)
	if err != nil {
		t.Fatalf("ReadEntry() with second identity error = %v", err)
	}
	if entry.Data["password"] != "StrongP@ssw0rd123" {
		t.Errorf("password = %v, want StrongP@ssw0rd123", entry.Data["password"])
	}
}

func TestHandleSet_MergePreservesMultiRecipientAccess(t *testing.T) {
	writerDir := t.TempDir()
	writerIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate writer identity: %v", err)
	}

	cfg := &config.Config{
		DefaultAgent: "test",
		Agents: map[string]config.AgentProfile{
			"test": {
				Name:         "test",
				AllowedPaths: []string{"*"},
				CanWrite:     true,
				ApprovalMode: "none",
			},
		},
	}
	if initErr := vault.Init(writerDir, writerIdentity, cfg); initErr != nil {
		t.Fatalf("vault.Init() error = %v", initErr)
	}

	existingEntry := &vault.Entry{
		Data: map[string]any{
			"username": "testuser",
		},
	}
	if writeErr := vault.WriteEntry(writerDir, "existing-entry", existingEntry, writerIdentity); writeErr != nil {
		t.Fatalf("WriteEntry() error = %v", writeErr)
	}

	secondIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate second identity: %v", err)
	}

	rm := vault.NewRecipientsManager(writerDir)
	if addErr := rm.AddRecipient(secondIdentity.Recipient().String()); addErr != nil {
		t.Fatalf("AddRecipient() error = %v", addErr)
	}

	srv := &Server{
		vault: &vault.Vault{
			Dir:      writerDir,
			Identity: writerIdentity,
		},
		agent: &config.AgentProfile{
			Name:         "test",
			AllowedPaths: []string{"*"},
			CanWrite:     true,
			ApprovalMode: "none",
		},
	}

	req := CallToolRequest{
		Arguments: map[string]any{
			"path":  "existing-entry",
			"field": "password",
			"value": "StrongP@ssw0rd123",
		},
	}
	_, err = srv.handleSet(context.Background(), req)
	if err != nil {
		t.Fatalf("handleSet() error = %v", err)
	}

	entry, err := vault.ReadEntry(writerDir, "existing-entry", secondIdentity)
	if err != nil {
		t.Fatalf("ReadEntry() with second identity error = %v", err)
	}
	if entry.Data["password"] != "StrongP@ssw0rd123" {
		t.Errorf("password = %v, want StrongP@ssw0rd123", entry.Data["password"])
	}
	if entry.Data["username"] != "testuser" {
		t.Errorf("username = %v, want testuser", entry.Data["username"])
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
