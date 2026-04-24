package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/testutil"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func TestOutputHermesStdioConfig_Success(t *testing.T) {
	output := captureStdout(func() {
		err := outputHermesStdioConfig("claude-code", "openpass")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(output, "mcp_servers") {
		t.Errorf("expected mcp_servers in output, got: %s", output)
	}
	if !strings.Contains(output, "openpass") {
		t.Errorf("expected server name in output, got: %s", output)
	}
	if !strings.Contains(output, "timeout") {
		t.Errorf("expected timeout in output, got: %s", output)
	}
}

func TestOutputHermesStdioConfig_StdoutError(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	_ = w.Close()
	_ = r.Close()
	defer func() { os.Stdout = oldStdout }()

	err := outputHermesStdioConfig("claude-code", "openpass")
	if err == nil {
		t.Fatal("expected error when stdout is closed")
	}
}

func TestOutputAgentStdioConfig_Success(t *testing.T) {
	output := captureStdout(func() {
		err := outputAgentStdioConfig("claude-code", "claude_code", "claude-code")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(output, "mcp_servers") {
		t.Errorf("expected mcp_servers in output, got: %s", output)
	}
	if !strings.Contains(output, "claude_code") {
		t.Errorf("expected server key in output, got: %s", output)
	}
}

func TestOutputAgentStdioConfig_StdoutError(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	_ = w.Close()
	_ = r.Close()
	defer func() { os.Stdout = oldStdout }()

	err := outputAgentStdioConfig("claude-code", "claude_code", "claude-code")
	if err == nil {
		t.Fatal("expected error when stdout is closed")
	}
}

func TestOutputAgentHTTPConfig_Success(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	output := captureStdout(func() {
		err := outputAgentHTTPConfig("claude-code", "claude_code", "claude-code", true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(output, "mcp_servers") {
		t.Errorf("expected mcp_servers in output, got: %s", output)
	}
	if !strings.Contains(output, "env:OPENPASS_MCP_TOKEN") {
		t.Errorf("expected redacted token in output, got: %s", output)
	}
}

func TestOutputAgentHTTPConfig_ResolveError(t *testing.T) {
	origHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	origVault := vault
	vault = "~/.openpass"
	defer func() { vault = origVault }()

	err := outputAgentHTTPConfig("claude-code", "claude_code", "claude-code", true)
	if err == nil {
		t.Fatal("expected error when vault path cannot be resolved")
	}
}

func TestOutputTokenOnly_Success(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	output := captureStdout(func() {
		err := outputTokenOnly()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if strings.TrimSpace(output) == "" {
		t.Error("expected token output, got empty string")
	}
}

func TestOutputTokenOnly_VaultPathError(t *testing.T) {
	origHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	origVault := vault
	vault = "~/.openpass"
	defer func() { vault = origVault }()

	err := outputTokenOnly()
	if err == nil {
		t.Fatal("expected error when vault path cannot be resolved")
	}
}

func TestOutputTokenOnly_CustomTokenPath(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	customTokenPath := filepath.Join(t.TempDir(), "custom-token")
	customToken := "my-custom-token-12345"
	if err := os.WriteFile(customTokenPath, []byte(customToken+"\n"), 0o600); err != nil {
		t.Fatalf("write custom token: %v", err)
	}

	cfgPath := filepath.Join(vaultDir, "config.yaml")
	configContent := "mcp:\n  httpTokenFile: " + customTokenPath + "\n"
	if err := os.WriteFile(cfgPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	output := captureStdout(func() {
		err := outputTokenOnly()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if strings.TrimSpace(output) != customToken {
		t.Errorf("expected token %q, got %q", customToken, strings.TrimSpace(output))
	}
}

func TestOutputHermesHTTPConfig_Success(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	output := captureStdout(func() {
		err := outputHermesHTTPConfig("claude-code", "openpass", true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(output, "mcp_servers") {
		t.Errorf("expected mcp_servers in output, got: %s", output)
	}
}

func TestOutputStdioConfig_Success(t *testing.T) {
	output := captureStdout(func() {
		err := outputStdioConfig("claude-code", "openpass")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(output, "mcpServers") {
		t.Errorf("expected mcpServers in output, got: %s", output)
	}
}
