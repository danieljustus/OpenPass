package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/mcp"
	"github.com/danieljustus/OpenPass/internal/testutil"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func TestMCPTokenCommandRegistration(t *testing.T) {
	commands := mcpTokenCmd.Commands()
	if len(commands) != 3 {
		t.Fatalf("expected 3 subcommands, got %d", len(commands))
	}

	names := make(map[string]bool)
	for _, c := range commands {
		names[c.Name()] = true
	}

	for _, want := range []string{"create", "list", "revoke"} {
		if !names[want] {
			t.Errorf("missing subcommand: %s", want)
		}
	}
}

func TestMCPTokenCreate_Defaults(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "create", "--label", "test-default"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Token created successfully.") {
		t.Errorf("expected success message, got: %s", output)
	}
	if !strings.Contains(output, "Raw token (copy now") {
		t.Errorf("expected raw token warning, got: %s", output)
	}
	if !strings.Contains(output, "test-default") {
		t.Errorf("expected label in output, got: %s", output)
	}
	if !strings.Contains(output, "Tools: *") {
		t.Errorf("expected wildcard tools in output, got: %s", output)
	}
}

func TestMCPTokenCreate_WithToolsAndAgent(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--tools", "list_entries,get_entry",
		"--agent", "claude-code",
		"--label", "scoped-test",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "scoped-test") {
		t.Errorf("expected label in output, got: %s", output)
	}
	if !strings.Contains(output, "Agent: claude-code") {
		t.Errorf("expected agent in output, got: %s", output)
	}
	if !strings.Contains(output, "Tools: list_entries, get_entry") {
		t.Errorf("expected tools in output, got: %s", output)
	}
}

func TestMCPTokenCreate_MultipleToolFlags(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--tools", "list_entries",
		"--tools", "get_entry",
		"--label", "multi-flag",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "multi-flag") {
		t.Errorf("expected label in output, got: %s", output)
	}
}

func TestMCPTokenCreate_WithTTL(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--ttl", "7d",
		"--label", "ttl-test",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "ttl-test") {
		t.Errorf("expected label in output, got: %s", output)
	}
	if !strings.Contains(output, "Expires:") {
		t.Errorf("expected expiration in output, got: %s", output)
	}
}

func TestMCPTokenCreate_DefaultTTLFromConfig(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	cfg.MCP = &config.MCPConfig{
		Bind: "127.0.0.1",
		Port: 8080,
	}
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--label", "config-ttl",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "config-ttl") {
		t.Errorf("expected label in output, got: %s", output)
	}
}

func TestMCPTokenCreate_InvalidTTL(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--ttl", "not-a-duration",
		"--label", "invalid",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected error for invalid TTL")
	}
	if !strings.Contains(execErr.Error(), "invalid TTL") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestMCPTokenCreate_VaultPathError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows: HOME env behavior differs")
	}
	origHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	origVault := vault
	vault = "~/.openpass"
	defer func() { vault = origVault }()

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"mcp", "token", "create", "--label", "fail"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected error when vault path cannot be resolved")
	}
}

func TestMCPTokenList_Empty(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "No tokens found.") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestMCPTokenList_WithTokens(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	reg := mcp.NewTokenRegistry(regPath)
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	_, _, err := reg.Create("list-test", []string{"list_entries", "get_entry"}, "claude-code", 24*time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "list-test") {
		t.Errorf("expected token label in output, got: %s", output)
	}
	if !strings.Contains(output, "claude-code") {
		t.Errorf("expected agent in output, got: %s", output)
	}
	if !strings.Contains(output, "active") {
		t.Errorf("expected status in output, got: %s", output)
	}
}

func TestMCPTokenRevoke_Success(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	reg := mcp.NewTokenRegistry(regPath)
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	token, _, err := reg.Create("revoke-test", []string{"*"}, "", 24*time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "revoke", token.ID})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "revoked successfully") {
		t.Errorf("expected revoke confirmation, got: %s", output)
	}

	reg2 := mcp.NewTokenRegistry(regPath)
	if err := reg2.Load(); err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	tokens := reg2.List()
	var found bool
	for i := range tokens {
		if tokens[i].ID == token.ID {
			found = true
			if !tokens[i].Revoked {
				t.Error("token should be revoked")
			}
		}
	}
	if !found {
		t.Error("revoked token should still be in list for audit")
	}
}

func TestMCPTokenRevoke_NotFound(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "revoke", "nonexistent-id"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected error for nonexistent token")
	}
	if !strings.Contains(execErr.Error(), "not found") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestMCPTokenRevoke_DoubleRevoke(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	reg := mcp.NewTokenRegistry(regPath)
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	token, _, err := reg.Create("double-revoke", []string{"*"}, "", 0)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "revoke", token.ID})
	var execErr error
	captureStdout(func() {
		execErr = rootCmd.Execute()
	})
	if execErr != nil {
		t.Fatalf("first revoke unexpected error: %v", execErr)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "revoke", token.ID})
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})
	if execErr == nil {
		t.Fatal("expected error for double revoke")
	}
	if !strings.Contains(execErr.Error(), "not found") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestMCPTokenList_RevokedToken(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	reg := mcp.NewTokenRegistry(regPath)
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	token, _, err := reg.Create("revoked-list", []string{"*"}, "", 0)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	reg.Revoke(token.ID)
	if err := reg.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "revoked") {
		t.Errorf("expected revoked status in output, got: %s", output)
	}
}

func TestParseHumanDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"24h", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"0h", 0, false},
		{"", 0, true},
		{"not-a-duration", 0, true},
		{"-1h", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseHumanDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHumanDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseHumanDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveTokenTTL_FromFlag(t *testing.T) {
	vaultDir := t.TempDir()
	d, err := resolveTokenTTL(vaultDir, "12h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 12*time.Hour {
		t.Errorf("ttl = %v, want 12h", d)
	}
}

func TestResolveTokenTTL_FromConfig(t *testing.T) {
	vaultDir := t.TempDir()
	d, err := resolveTokenTTL(vaultDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", d)
	}
}

func TestResolveTokenTTL_DefaultFallback(t *testing.T) {
	vaultDir := t.TempDir()
	d, err := resolveTokenTTL(vaultDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", d)
	}
}

func TestMCPTokenCreate_ZeroTTL(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--ttl", "0h",
		"--label", "zero-ttl",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Expires: never") {
		t.Errorf("expected 'never' expiration for zero TTL, got: %s", output)
	}
}

func TestMCPTokenList_ExpiredTokenExcluded(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	reg := mcp.NewTokenRegistry(regPath)
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	_, _, err := reg.Create("expired", []string{"*"}, "", 1*time.Nanosecond)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	_, _, err = reg.Create("valid", []string{"*"}, "", 1*time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if strings.Contains(output, "expired") {
		t.Errorf("expired token should not appear in list, got: %s", output)
	}
	if !strings.Contains(output, "valid") {
		t.Errorf("valid token should appear in list, got: %s", output)
	}
}

func TestMCPTokenCreate_PreservesInRegistry(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--tools", "list_entries",
		"--label", "persist-test",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	data, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if !strings.Contains(string(data), "persist-test") {
		t.Errorf("registry should contain token label, got: %s", string(data))
	}
	if strings.Contains(string(data), "Raw token") {
		t.Error("registry should not contain raw token")
	}
}

func TestMCPTokenRevoke_MissingArg(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "revoke"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestMCPTokenCreate_ToolsLongOutput(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	manyTools := []string{
		"tool_a", "tool_b", "tool_c", "tool_d", "tool_e",
		"tool_f", "tool_g", "tool_h", "tool_i", "tool_j",
	}

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	reg := mcp.NewTokenRegistry(regPath)
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	_, _, err := reg.Create("many-tools", manyTools, "", 0)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "many-tools") {
		t.Errorf("expected token in output, got: %s", output)
	}
}

func TestMCPTokenList_HeaderFormat(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	reg := mcp.NewTokenRegistry(regPath)
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	_, _, err := reg.Create("header-test", []string{"*"}, "agent-name", 0)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	headers := []string{"ID", "LABEL", "AGENT", "TOOLS", "EXPIRES AT", "STATUS"}
	for _, h := range headers {
		if !strings.Contains(output, h) {
			t.Errorf("expected header %q in output, got: %s", h, output)
		}
	}
}

func TestMCPCmdRegistration(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "mcp" {
			found = true
			break
		}
	}
	if !found {
		t.Error("mcp command not registered under root")
	}
}

func TestMCPTokenCreate_NegativeDayTTL(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--ttl", "-1d",
		"--label", "negative",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected error for negative TTL")
	}
	if !strings.Contains(execErr.Error(), "invalid TTL") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestMCPTokenCreate_RegistryFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows: file permissions differ")
	}
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--label", "perms-test",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	info, err := os.Stat(regPath)
	if err != nil {
		t.Fatalf("stat registry: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("registry file permissions = %o, want 600", perm)
	}
}

func TestMCPTokenCreate_EmptyLabelOK(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Token created successfully.") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestMCPTokenRevoke_VaultPathError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows: HOME env behavior differs")
	}
	origHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	origVault := vault
	vault = "~/.openpass"
	defer func() { vault = origVault }()

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"mcp", "token", "revoke", "some-id"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected error when vault path cannot be resolved")
	}
}

func TestMCPTokenList_VaultPathError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows: HOME env behavior differs")
	}
	origHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	defer func() { _ = os.Setenv("HOME", origHome) }()

	origVault := vault
	vault = "~/.openpass"
	defer func() { vault = origVault }()

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"mcp", "token", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected error when vault path cannot be resolved")
	}
}

func TestMCPTokenCreate_WithDaySuffixTTL(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{
		"--vault", vaultDir,
		"mcp", "token", "create",
		"--ttl", "7d",
		"--label", "week-token",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "week-token") {
		t.Errorf("expected label in output, got: %s", output)
	}
	if !strings.Contains(output, "Expires:") {
		t.Errorf("expected expiration in output, got: %s", output)
	}
}

func TestMCPTokenList_EmptyAgentAndLabel(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	regPath := mcp.TokenRegistryFilePath(vaultDir)
	reg := mcp.NewTokenRegistry(regPath)
	if err := reg.Load(); err != nil {
		t.Fatalf("load registry: %v", err)
	}
	_, _, err := reg.Create("", []string{"*"}, "", 0)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	vaultFlagReset(t)
	t.Cleanup(func() { resetCobraCommand(rootCmd) })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp", "token", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	t.Cleanup(func() { _ = tokenCreateCmd.Flags().Set("ttl", "") })

	output := captureStdout(func() {
		err := rootCmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "-") {
		t.Errorf("expected dash for empty label/agent, got: %s", output)
	}
}

func TestMCPTokenCreate_RawTokenUnique(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	var tokens []string
	for i := 0; i < 3; i++ {
		vaultFlagReset(t)
		t.Cleanup(func() { resetCobraCommand(rootCmd) })

		rootCmd.SetArgs([]string{
			"--vault", vaultDir,
			"mcp", "token", "create",
			"--label", fmt.Sprintf("token-%d", i),
		})

		output := captureStdout(func() {
			err := rootCmd.Execute()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
		rootCmd.SetArgs(nil)

		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "Raw token (copy now") {
				parts := strings.Split(line, ": ")
				if len(parts) == 2 {
					tokens = append(tokens, strings.TrimSpace(parts[1]))
				}
			}
		}
	}

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}
	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens); j++ {
			if tokens[i] == tokens[j] {
				t.Errorf("tokens %d and %d are identical: %s", i, j, tokens[i])
			}
		}
	}
}
