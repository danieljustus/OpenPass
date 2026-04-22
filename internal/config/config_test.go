package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultReturnsSensibleConfig(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if cfg == nil {
		t.Fatal("Default returned nil")
	}

	wantVaultDir := filepath.Join(mustHomeDir(t), ".openpass")
	if cfg.VaultDir != wantVaultDir {
		t.Fatalf("VaultDir = %q, want %q", cfg.VaultDir, wantVaultDir)
	}
	if cfg.DefaultAgent != "default" {
		t.Fatalf("DefaultAgent = %q, want %q", cfg.DefaultAgent, "default")
	}

	// Built-in profile assertions
	type wantProfile struct {
		approvalMode string
		canWrite     bool
	}
	wantProfiles := map[string]wantProfile{
		"default":     {canWrite: false, approvalMode: "none"},
		"claude-code": {canWrite: true, approvalMode: "none"},
		"codex":       {canWrite: false, approvalMode: "none"},
		"hermes":      {canWrite: true, approvalMode: "none"},
		"openclaw":    {canWrite: true, approvalMode: "none"},
		"opencode":    {canWrite: false, approvalMode: "none"},
	}
	for name, want := range wantProfiles {
		got, ok := cfg.Agents[name]
		if !ok {
			t.Fatalf("missing built-in profile: %s", name)
		}
		if got.CanWrite != want.canWrite {
			t.Fatalf("profile %q CanWrite = %v, want %v", name, got.CanWrite, want.canWrite)
		}
		if got.ApprovalMode != want.approvalMode {
			t.Fatalf("profile %q ApprovalMode = %q, want %q", name, got.ApprovalMode, want.approvalMode)
		}
	}
}

func TestLoadUsesDefaultsForMissingFields(t *testing.T) {
	t.Parallel()

	path := writeTempFile(t, []byte("vaultDir: /custom/vault\nagents:\n  claude:\n    allowedPaths:\n      - personal/\n    canWrite: false\n    approvalMode: prompt\n"))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.VaultDir != "/custom/vault" {
		t.Fatalf("VaultDir = %q, want %q", cfg.VaultDir, "/custom/vault")
	}
	if _, ok := cfg.Agents["default"]; !ok {
		t.Fatal("default profile should be present when omitted from file")
	}

	want := AgentProfile{
		Name:         "claude",
		AllowedPaths: []string{"personal/"},
		CanWrite:     false,
		ApprovalMode: "prompt",
	}
	if got := cfg.Agents["claude"]; got.Name != want.Name || got.CanWrite != want.CanWrite || got.ApprovalMode != want.ApprovalMode {
		t.Fatalf("claude profile = %+v, want %+v", got, want)
	}
}

func TestLoadEmptyFileReturnsDefaults(t *testing.T) {
	t.Parallel()

	path := writeTempFile(t, nil)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DefaultAgent != "default" {
		t.Fatalf("DefaultAgent = %q, want %q", cfg.DefaultAgent, "default")
	}
	if _, ok := cfg.Agents["default"]; !ok {
		t.Fatal("missing default agent profile")
	}
}

func TestLoadMissingFileReturnsError(t *testing.T) {
	t.Parallel()

	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadRejectsInvalidYAML(t *testing.T) {
	t.Parallel()

	path := writeTempFile(t, []byte("vaultDir: [unterminated\n"))
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestSaveWritesToDefaultConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		VaultDir:       filepath.Join(home, ".openpass"),
		DefaultAgent:   "default",
		SessionTimeout: defaultSessionTimeout,
		Agents: map[string]AgentProfile{
			"default": {
				Name:         "default",
				AllowedPaths: []string{},
				CanWrite:     false,
				ApprovalMode: "none",
			},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	wantPath := filepath.Join(home, ".openpass", "config.yaml")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("config file missing at %q: %v", wantPath, err)
	}

	loaded, err := Load(wantPath)
	if err != nil {
		t.Fatalf("Load(saved) error = %v", err)
	}
	agent := loaded.Agents["default"]
	if agent.CanWrite != false || agent.ApprovalMode != "none" {
		t.Fatalf("saved agent = %+v, want CanWrite=false ApprovalMode=none", agent)
	}
}

func TestSaveCreatesConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := Default()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".openpass")); err != nil {
		t.Fatalf("config directory missing: %v", err)
	}
}

func TestLoadParsesApprovalMode(t *testing.T) {
	t.Parallel()

	yaml := "agents:\n  myagent:\n    allowedPaths: [\"*\"]\n    canWrite: true\n    approvalMode: deny\n"
	path := writeTempFile(t, []byte(yaml))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	agent := cfg.Agents["myagent"]
	if agent.ApprovalMode != "deny" {
		t.Fatalf("ApprovalMode = %q, want %q", agent.ApprovalMode, "deny")
	}
}

func TestLoadMapsRequireApprovalToApprovalMode(t *testing.T) {
	t.Parallel()

	yaml := "agents:\n  old-agent:\n    allowedPaths: [\"*\"]\n    canWrite: true\n    requireApproval: true\n"
	path := writeTempFile(t, []byte(yaml))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	agent := cfg.Agents["old-agent"]
	if agent.ApprovalMode != "prompt" {
		t.Fatalf("ApprovalMode = %q, want %q", agent.ApprovalMode, "prompt")
	}
}

func TestLoadApprovalModeTakesPrecedence(t *testing.T) {
	t.Parallel()

	yaml := "agents:\n  both:\n    allowedPaths: [\"*\"]\n    canWrite: true\n    requireApproval: true\n    approvalMode: none\n"
	path := writeTempFile(t, []byte(yaml))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	agent := cfg.Agents["both"]
	if agent.ApprovalMode != "none" {
		t.Fatalf("ApprovalMode = %q, want %q", agent.ApprovalMode, "none")
	}
}

func TestLoadRejectsInvalidApprovalMode(t *testing.T) {
	t.Parallel()

	yaml := "agents:\n  bad:\n    approvalMode: invalid\n"
	path := writeTempFile(t, []byte(yaml))

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid approvalMode, got nil")
	}
}

func TestLoadParsesRedactFields(t *testing.T) {
	t.Parallel()

	yaml := "agents:\n  restricted:\n    allowedPaths: [\"*\"]\n    canWrite: false\n    redactFields:\n      - totp.secret\n      - password\n      - api.*\n"
	path := writeTempFile(t, []byte(yaml))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	agent := cfg.Agents["restricted"]
	if len(agent.RedactFields) != 3 {
		t.Fatalf("RedactFields length = %d, want 3", len(agent.RedactFields))
	}
	if agent.RedactFields[0] != "totp.secret" {
		t.Errorf("RedactFields[0] = %q, want %q", agent.RedactFields[0], "totp.secret")
	}
	if agent.RedactFields[1] != "password" {
		t.Errorf("RedactFields[1] = %q, want %q", agent.RedactFields[1], "password")
	}
	if agent.RedactFields[2] != "api.*" {
		t.Errorf("RedactFields[2] = %q, want %q", agent.RedactFields[2], "api.*")
	}
}

func TestSaveWritesRedactFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		VaultDir:       filepath.Join(home, ".openpass"),
		DefaultAgent:   "default",
		SessionTimeout: defaultSessionTimeout,
		Agents: map[string]AgentProfile{
			"restricted": {
				Name:         "restricted",
				AllowedPaths: []string{"*"},
				CanWrite:     false,
				ApprovalMode: "deny",
				RedactFields: []string{"totp.secret", "password"},
			},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(filepath.Join(home, ".openpass", "config.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	agent := loaded.Agents["restricted"]
	if len(agent.RedactFields) != 2 {
		t.Fatalf("RedactFields length = %d, want 2", len(agent.RedactFields))
	}
	if agent.RedactFields[0] != "totp.secret" || agent.RedactFields[1] != "password" {
		t.Errorf("RedactFields = %v, want [totp.secret password]", agent.RedactFields)
	}
}

func TestNewDefaultAgentProfile(t *testing.T) {
	t.Parallel()

	profile := newDefaultAgentProfile("test-agent")

	if profile.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", profile.Name, "test-agent")
	}
	if profile.AllowedPaths == nil {
		t.Fatal("AllowedPaths should not be nil")
	}
	if len(profile.AllowedPaths) != 0 {
		t.Errorf("AllowedPaths length = %d, want 0", len(profile.AllowedPaths))
	}
	if profile.CanWrite {
		t.Error("CanWrite should be false")
	}
	if profile.ApprovalMode != "none" {
		t.Errorf("ApprovalMode = %q, want %q", profile.ApprovalMode, "none")
	}
}

func TestLoadWithVaultGitMCPSections(t *testing.T) {
	t.Parallel()

	yaml := `vault:
  path: /my/vault
  default_recipients:
    - age1abc
git:
  auto_push: false
  commit_template: "custom commit"
mcp:
  port: 9090
  bind: "0.0.0.0"
clipboard:
  auto_clear_duration: 60
`
	path := writeTempFile(t, []byte(yaml))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Vault == nil {
		t.Fatal("Vault should not be nil")
	}
	if cfg.Vault.Path != "/my/vault" {
		t.Errorf("Vault.Path = %q, want %q", cfg.Vault.Path, "/my/vault")
	}
	if len(cfg.Vault.DefaultRecipients) != 1 || cfg.Vault.DefaultRecipients[0] != "age1abc" {
		t.Errorf("Vault.DefaultRecipients = %v, want [age1abc]", cfg.Vault.DefaultRecipients)
	}

	if cfg.Git == nil {
		t.Fatal("Git should not be nil")
	}
	if cfg.Git.AutoPush {
		t.Error("Git.AutoPush should be false")
	}
	if cfg.Git.CommitTemplate != "custom commit" {
		t.Errorf("Git.CommitTemplate = %q, want %q", cfg.Git.CommitTemplate, "custom commit")
	}

	if cfg.MCP == nil {
		t.Fatal("MCP should not be nil")
	}
	if cfg.MCP.Port != 9090 {
		t.Errorf("MCP.Port = %d, want %d", cfg.MCP.Port, 9090)
	}
	if cfg.MCP.Bind != "0.0.0.0" {
		t.Errorf("MCP.Bind = %q, want %q", cfg.MCP.Bind, "0.0.0.0")
	}

	if cfg.Clipboard == nil {
		t.Fatal("Clipboard should not be nil")
	}
	if cfg.Clipboard.AutoClearDuration != 60 {
		t.Errorf("Clipboard.AutoClearDuration = %d, want %d", cfg.Clipboard.AutoClearDuration, 60)
	}
}

func TestLoadWithOnlyVaultSection(t *testing.T) {
	t.Parallel()

	yaml := `vault:
  path: /only/vault
`
	path := writeTempFile(t, []byte(yaml))

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Vault == nil {
		t.Fatal("Vault should not be nil")
	}
	if cfg.Vault.Path != "/only/vault" {
		t.Errorf("Vault.Path = %q, want %q", cfg.Vault.Path, "/only/vault")
	}
}

func TestSaveWithAllConfigSections(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		VaultDir:       filepath.Join(home, ".openpass"),
		DefaultAgent:   "default",
		SessionTimeout: defaultSessionTimeout,
		Agents: map[string]AgentProfile{
			"default": {
				Name:         "default",
				AllowedPaths: []string{"*"},
				CanWrite:     false,
				ApprovalMode: "none",
			},
		},
		Vault: &VaultConfig{
			Path:              "/vault/path",
			DefaultRecipients: []string{"recipient1"},
			ConfirmRemove:     true,
		},
		Git: &GitConfig{
			AutoPush:       false,
			CommitTemplate: "custom",
		},
		MCP: &MCPConfig{
			Port:          9090,
			Bind:          "0.0.0.0",
			Stdio:         true,
			HTTPTokenFile: "/token/path",
		},
		Clipboard: &ClipboardConfig{
			AutoClearDuration: 60,
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	wantPath := filepath.Join(home, ".openpass", "config.yaml")
	loaded, err := Load(wantPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Vault == nil {
		t.Fatal("Vault should be saved and loaded")
	}
	if loaded.Vault.Path != "/vault/path" {
		t.Errorf("Vault.Path = %q, want %q", loaded.Vault.Path, "/vault/path")
	}
	if loaded.Vault.DefaultRecipients[0] != "recipient1" {
		t.Errorf("Vault.DefaultRecipients = %v, want [recipient1]", loaded.Vault.DefaultRecipients)
	}

	if loaded.Git == nil {
		t.Fatal("Git should be saved and loaded")
	}
	if loaded.Git.AutoPush {
		t.Error("Git.AutoPush should be false")
	}

	if loaded.MCP == nil {
		t.Fatal("MCP should be saved and loaded")
	}
	if loaded.MCP.Port != 9090 {
		t.Errorf("MCP.Port = %d, want %d", loaded.MCP.Port, 9090)
	}

	if loaded.Clipboard == nil {
		t.Fatal("Clipboard should be saved and loaded")
	}
	if loaded.Clipboard.AutoClearDuration != 60 {
		t.Errorf("Clipboard.AutoClearDuration = %d, want %d", loaded.Clipboard.AutoClearDuration, 60)
	}
}

func TestSaveWithNilConfigReturnsError(t *testing.T) {
	var cfg *Config
	if err := cfg.Save(); err == nil {
		t.Fatal("Save() on nil Config should return error")
	}
}

func TestSaveLoadRoundTrip_PreservesAllFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{
		VaultDir:       filepath.Join(home, ".openpass"),
		DefaultAgent:   "test-agent",
		SessionTimeout: defaultSessionTimeout,
		Agents: map[string]AgentProfile{
			"test-agent": {
				Name:            "test-agent",
				AllowedPaths:    []string{"path1", "path2"},
				CanWrite:        true,
				ApprovalMode:    "prompt",
				ApprovalTimeout: 2 * time.Minute,
				RequireApproval: true,
			},
		},
		Vault: &VaultConfig{
			Path:              "/vault/path",
			DefaultRecipients: []string{"recipient1", "recipient2"},
			ConfirmRemove:     true,
		},
		MCP: &MCPConfig{
			Port:              9090,
			Bind:              "0.0.0.0",
			Stdio:             true,
			HTTPTokenFile:     "/token/path",
			ReadHeaderTimeout: 7 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      20 * time.Second,
			ShutdownTimeout:   8 * time.Second,
			ApprovalTimeout:   45 * time.Second,
		},
		Clipboard: &ClipboardConfig{
			AutoClearDuration: 90,
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	wantPath := filepath.Join(home, ".openpass", "config.yaml")
	loaded, err := Load(wantPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify Vault
	if loaded.Vault == nil {
		t.Fatal("Vault should not be nil after round-trip")
	}
	if loaded.Vault.Path != cfg.Vault.Path {
		t.Errorf("Vault.Path = %q, want %q", loaded.Vault.Path, cfg.Vault.Path)
	}
	if len(loaded.Vault.DefaultRecipients) != len(cfg.Vault.DefaultRecipients) {
		t.Errorf("Vault.DefaultRecipients len = %d, want %d", len(loaded.Vault.DefaultRecipients), len(cfg.Vault.DefaultRecipients))
	}
	if loaded.Vault.ConfirmRemove != cfg.Vault.ConfirmRemove {
		t.Errorf("Vault.ConfirmRemove = %v, want %v", loaded.Vault.ConfirmRemove, cfg.Vault.ConfirmRemove)
	}

	// Verify MCP
	if loaded.MCP == nil {
		t.Fatal("MCP should not be nil after round-trip")
	}
	if loaded.MCP.Port != cfg.MCP.Port {
		t.Errorf("MCP.Port = %d, want %d", loaded.MCP.Port, cfg.MCP.Port)
	}
	if loaded.MCP.Bind != cfg.MCP.Bind {
		t.Errorf("MCP.Bind = %q, want %q", loaded.MCP.Bind, cfg.MCP.Bind)
	}
	if loaded.MCP.Stdio != cfg.MCP.Stdio {
		t.Errorf("MCP.Stdio = %v, want %v", loaded.MCP.Stdio, cfg.MCP.Stdio)
	}
	if loaded.MCP.HTTPTokenFile != cfg.MCP.HTTPTokenFile {
		t.Errorf("MCP.HTTPTokenFile = %q, want %q", loaded.MCP.HTTPTokenFile, cfg.MCP.HTTPTokenFile)
	}
	if loaded.MCP.ReadHeaderTimeout != cfg.MCP.ReadHeaderTimeout {
		t.Errorf("MCP.ReadHeaderTimeout = %v, want %v", loaded.MCP.ReadHeaderTimeout, cfg.MCP.ReadHeaderTimeout)
	}
	if loaded.MCP.ReadTimeout != cfg.MCP.ReadTimeout {
		t.Errorf("MCP.ReadTimeout = %v, want %v", loaded.MCP.ReadTimeout, cfg.MCP.ReadTimeout)
	}
	if loaded.MCP.WriteTimeout != cfg.MCP.WriteTimeout {
		t.Errorf("MCP.WriteTimeout = %v, want %v", loaded.MCP.WriteTimeout, cfg.MCP.WriteTimeout)
	}
	if loaded.MCP.ShutdownTimeout != cfg.MCP.ShutdownTimeout {
		t.Errorf("MCP.ShutdownTimeout = %v, want %v", loaded.MCP.ShutdownTimeout, cfg.MCP.ShutdownTimeout)
	}
	if loaded.MCP.ApprovalTimeout != cfg.MCP.ApprovalTimeout {
		t.Errorf("MCP.ApprovalTimeout = %v, want %v", loaded.MCP.ApprovalTimeout, cfg.MCP.ApprovalTimeout)
	}

	// Verify Clipboard
	if loaded.Clipboard == nil {
		t.Fatal("Clipboard should not be nil after round-trip")
	}
	if loaded.Clipboard.AutoClearDuration != cfg.Clipboard.AutoClearDuration {
		t.Errorf("Clipboard.AutoClearDuration = %d, want %d", loaded.Clipboard.AutoClearDuration, cfg.Clipboard.AutoClearDuration)
	}

	// Verify AgentProfile
	agent, ok := loaded.Agents["test-agent"]
	if !ok {
		t.Fatal("test-agent profile should exist after round-trip")
	}
	if agent.CanWrite != cfg.Agents["test-agent"].CanWrite {
		t.Errorf("agent.CanWrite = %v, want %v", agent.CanWrite, cfg.Agents["test-agent"].CanWrite)
	}
	if agent.RequireApproval != cfg.Agents["test-agent"].RequireApproval {
		t.Errorf("agent.RequireApproval = %v, want %v", agent.RequireApproval, cfg.Agents["test-agent"].RequireApproval)
	}
	if agent.ApprovalTimeout != cfg.Agents["test-agent"].ApprovalTimeout {
		t.Errorf("agent.ApprovalTimeout = %v, want %v", agent.ApprovalTimeout, cfg.Agents["test-agent"].ApprovalTimeout)
	}
	if agent.ApprovalMode != cfg.Agents["test-agent"].ApprovalMode {
		t.Errorf("agent.ApprovalMode = %q, want %q", agent.ApprovalMode, cfg.Agents["test-agent"].ApprovalMode)
	}
	if len(agent.AllowedPaths) != len(cfg.Agents["test-agent"].AllowedPaths) {
		t.Errorf("agent.AllowedPaths len = %d, want %d", len(agent.AllowedPaths), len(cfg.Agents["test-agent"].AllowedPaths))
	}
}

func mustHomeDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() error = %v", err)
	}
	return home
}

func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
