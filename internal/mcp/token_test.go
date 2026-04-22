package mcp

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateTokenCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-token")

	token, err := LoadOrCreateToken(path)
	if err != nil {
		t.Fatalf("LoadOrCreateToken() error = %v", err)
	}
	if len(token) == 0 {
		t.Fatal("token is empty")
	}

	token2, err := LoadOrCreateToken(path)
	if err != nil {
		t.Fatalf("second LoadOrCreateToken() error = %v", err)
	}
	if token != token2 {
		t.Fatalf("token changed on second load: %q vs %q", token, token2)
	}
}

func TestLoadOrCreateTokenRespectsEnvVar(t *testing.T) {
	t.Setenv("OPENPASS_MCP_TOKEN", "my-custom-token")

	token, err := LoadOrCreateToken("/nonexistent/path")
	if err != nil {
		t.Fatalf("LoadOrCreateToken() error = %v", err)
	}
	if token != "my-custom-token" {
		t.Fatalf("token = %q, want %q", token, "my-custom-token")
	}
}

func TestLoadOrCreateTokenFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-token")

	_, err := LoadOrCreateToken(path)
	if err != nil {
		t.Fatalf("LoadOrCreateToken() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file permissions = %o, want 600", perm)
	}
}

func TestLoadOrCreateToken_WhitespaceFile_GeneratesNewToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-token")

	if err := os.WriteFile(path, []byte("   \n\t  \n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	token, err := LoadOrCreateToken(path)
	if err != nil {
		t.Fatalf("LoadOrCreateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("expected generated token, got empty string")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != token+"\n" {
		t.Fatalf("file content = %q, want %q", string(data), token+"\n")
	}
}

func TestLoadOrCreateToken_FileTokenIgnoresEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-token")

	if err := os.WriteFile(path, []byte("file-token\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("OPENPASS_MCP_TOKEN", "env-token")

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = w

	token, err := LoadOrCreateToken(path)

	os.Stderr = oldStderr
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("LoadOrCreateToken() error = %v", err)
	}
	if token != "file-token" {
		t.Fatalf("token = %q, want %q", token, "file-token")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Warning: OPENPASS_MCP_TOKEN is set")) {
		t.Fatalf("expected stderr warning, got %q", buf.String())
	}
}

func TestLoadOrCreateToken_RandError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-token")

	oldReader := randReader
	randReader = &errorReader{}
	defer func() { randReader = oldReader }()

	_, err := LoadOrCreateToken(path)
	if err == nil {
		t.Fatal("expected error from rand.Reader failure")
	}
}

func TestLoadOrCreateToken_WriteFileError(t *testing.T) {
	t.Setenv("OPENPASS_MCP_TOKEN", "")
	path := filepath.Join("/nonexistent-dir-openpass-test", "mcp-token")
	_, err := LoadOrCreateToken(path)
	if err == nil {
		t.Fatal("expected error from WriteFile failure")
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("rand failure")
}

func TestRotateTokenCreatesNewToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-token")

	if err := os.WriteFile(path, []byte("old-token\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	newToken, err := RotateToken(path)
	if err != nil {
		t.Fatalf("RotateToken() error = %v", err)
	}
	if newToken == "" {
		t.Fatal("new token is empty")
	}
	if newToken == "old-token" {
		t.Fatal("RotateToken should have generated a new token, not returned the old one")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != newToken+"\n" {
		t.Fatalf("file content = %q, want %q", string(data), newToken+"\n")
	}
}

func TestRotateTokenSetsCorrectPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-token")

	_, err := RotateToken(path)
	if err != nil {
		t.Fatalf("RotateToken() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file permissions = %o, want 600", perm)
	}
}

func TestRotateTokenGeneratesDifferentTokens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp-token")

	token1, err := RotateToken(path)
	if err != nil {
		t.Fatalf("RotateToken() error = %v", err)
	}

	token2, err := RotateToken(path)
	if err != nil {
		t.Fatalf("RotateToken() error = %v", err)
	}

	if token1 == token2 {
		t.Fatal("consecutive RotateToken calls should generate different tokens")
	}
}
