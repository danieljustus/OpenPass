package cmd

import (
	"fmt"
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

func TestCmdMCPConfig_HTTPMode(t *testing.T) {
	vaultDir, _ := initVault(t)
	defer setupVaultFlag(t, vaultDir)()
	cfgPath := filepath.Join(vaultDir, "config.yaml")
	cfg := "mcp:\n  bind: 127.0.0.1\n  port: 9999\n  http_token_file: \"\"\n"
	_ = os.WriteFile(cfgPath, []byte(cfg), 0o600)
	_ = execWithStdout("--vault", vaultDir, "mcp-config", "default", "--http")
}

func TestCmdMCPConfig_Stdio(t *testing.T) {
	vaultFlagReset(t)
	_ = mcpConfigCmd.Flags().Set("http", "false")
	t.Cleanup(func() { _ = mcpConfigCmd.Flags().Set("http", "false") })

	rootCmd.SetArgs([]string{"mcp-config", "myagent"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "openpass") {
		t.Errorf("mcp-config stdio output missing 'openpass': %q", output)
	}
	if !strings.Contains(output, "myagent") {
		t.Errorf("mcp-config stdio output missing agent name: %q", output)
	}
}

func TestCmdMCPConfig_StdioCustomVaultIncludesVaultArg(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "--vault") || !strings.Contains(output, vaultDir) {
		t.Errorf("mcp-config stdio output for custom vault missing --vault arg: %q", output)
	}
}

func TestCmdMCPConfig_HTTP(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_VAULT") })

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "url") {
		t.Errorf("mcp-config http output missing 'url': %q", output)
	}
	if !strings.Contains(output, "Authorization") {
		t.Errorf("mcp-config http output missing 'Authorization': %q", output)
	}
}

func TestCmdMCPConfig_HermesHTTP(t *testing.T) {
	vaultDir, _ := initVault(t)
	vaultFlagReset(t)
	_ = mcpConfigCmd.Flags().Set("format", "generic")
	_ = mcpConfigCmd.Flags().Set("server-name", "openpass")
	t.Cleanup(func() {
		_ = mcpConfigCmd.Flags().Set("format", "generic")
		_ = mcpConfigCmd.Flags().Set("server-name", "openpass")
	})

	cfgContent := "mcp:\n  bind: 127.0.0.1\n  port: 8090\n"
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "hermes", "--http", "--format", "hermes"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	for _, want := range []string{
		"mcp_servers:",
		"openpass:",
		"url: http://127.0.0.1:8090/mcp",
		"Authorization: env:OPENPASS_MCP_TOKEN",
		`MCP-Protocol-Version: "2025-11-25"`,
		"X-OpenPass-Agent: hermes",
		"connect_timeout: 30",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("mcp-config hermes http output missing %q: %q", want, output)
		}
	}
}

func TestOutputHTTPConfig_VaultPathError(t *testing.T) {
	// Test outputHTTPConfig directly (bypassing rootCmd which has PersistentPreRun
	// that also calls vaultPath, causing a panic before our function is reached).
	origHome := os.Getenv("HOME")
	origVaultEnv := os.Getenv("OPENPASS_VAULT")
	origVault := vault
	origChanged := vaultFlag.Changed
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	vault = "~/.openpass"
	vaultFlag.Changed = false
	t.Cleanup(func() {
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("OPENPASS_VAULT", origVaultEnv)
		vault = origVault
		_ = vaultFlag.Value.Set(origVault)
		vaultFlag.Changed = origChanged
	})

	err := outputHTTPConfig("test-agent", "openpass", true)
	if err == nil {
		t.Error("expected error when HOME is unset for tilde expansion")
	}
}

func TestOutputHTTPConfig_CustomTokenFile(t *testing.T) {
	vaultDir, _ := initVault(t)
	vaultFlagReset(t)
	t.Cleanup(func() { _ = mcpConfigCmd.Flags().Set("include-token", "false") })

	customTokenPath := filepath.Join(t.TempDir(), "custom-token")
	customTokenValue := "my-custom-token-value-12345"
	if err := os.WriteFile(customTokenPath, []byte(customTokenValue+"\n"), 0o600); err != nil {
		t.Fatalf("write custom token: %v", err)
	}

	cfgContent := fmt.Sprintf("mcp:\n  bind: 127.0.0.1\n  port: 9999\n  httpTokenFile: %q\n", customTokenPath)
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "127.0.0.1:9999") {
		t.Errorf("mcp-config http output missing custom bind/port: %q", output)
	}
	if strings.Contains(output, customTokenValue) {
		t.Errorf("mcp-config http output leaked custom token without --include-token: %q", output)
	}
	if !strings.Contains(output, "env:OPENPASS_MCP_TOKEN") {
		t.Errorf("mcp-config http output missing redacted token reference: %q", output)
	}
	if !strings.Contains(output, "Authorization") {
		t.Errorf("mcp-config http output missing 'Authorization': %q", output)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http", "--include-token"})
	output = captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, customTokenValue) {
		t.Errorf("mcp-config http --include-token output missing custom token: %q", output)
	}
}

func TestOutputHTTPConfig_TokenLoadError(t *testing.T) {
	vaultDir, _ := initVault(t)
	vaultFlagReset(t)

	cfgContent := "mcp:\n  httpTokenFile: /nonexistent/path/mcp-token\n"
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for non-existent token path")
	}
	if !strings.Contains(execErr.Error(), "load token") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestOutputHTTPConfig_StaleRuntimePort(t *testing.T) {
	vaultDir, _ := initVault(t)
	vaultFlagReset(t)

	if err := saveRuntimePort(vaultDir, 1); err != nil {
		t.Fatalf("save runtime port: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected stale runtime port error")
	}
	if !strings.Contains(execErr.Error(), "stale runtime port") {
		t.Fatalf("unexpected error: %v", execErr)
	}
}
