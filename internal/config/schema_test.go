package config

import (
	"testing"
)

func TestDefaultVaultConfig(t *testing.T) {
	cfg := defaultVaultConfig()

	if cfg.Path != "" {
		t.Errorf("Path should be empty, got %q", cfg.Path)
	}
	if cfg.DefaultRecipients == nil {
		t.Fatal("DefaultRecipients should be initialized")
	}
	if len(cfg.DefaultRecipients) != 0 {
		t.Errorf("DefaultRecipients should be empty, got %v", cfg.DefaultRecipients)
	}
}

func TestDefaultGitConfig(t *testing.T) {
	cfg := defaultGitConfig()

	if !cfg.AutoPush {
		t.Error("AutoPush should be true by default")
	}
	if cfg.CommitTemplate != "Update from OpenPass" {
		t.Errorf("CommitTemplate mismatch: got %q, want %q", cfg.CommitTemplate, "Update from OpenPass")
	}
}

func TestDefaultMCPConfig(t *testing.T) {
	cfg := defaultMCPConfig()

	if cfg.Port != 8080 {
		t.Errorf("Port should be 8080, got %d", cfg.Port)
	}
	if cfg.Bind != "127.0.0.1" {
		t.Errorf("Bind should be 127.0.0.1, got %q", cfg.Bind)
	}
	if cfg.Stdio {
		t.Error("Stdio should be false by default")
	}
	if cfg.HTTPTokenFile != "auto" {
		t.Errorf("HTTPTokenFile should be auto, got %q", cfg.HTTPTokenFile)
	}
	if cfg.ApprovalRequired {
		t.Error("ApprovalRequired should be false (deprecated, not set by default)")
	}
}

func TestMergeFileVaultConfig(t *testing.T) {
	defaults := defaultVaultConfig()

	t.Run("nil file config returns defaults", func(t *testing.T) {
		result := MergeFileVaultConfig(nil, defaults)
		if result.Path != defaults.Path {
			t.Error("Path should be default")
		}
	})

	t.Run("file config overrides defaults", func(t *testing.T) {
		fileCfg := &fileVaultConfig{
			Path:              "/custom/path",
			DefaultRecipients: []string{"recipient1", "recipient2"},
		}
		result := MergeFileVaultConfig(fileCfg, defaults)

		if result.Path != "/custom/path" {
			t.Errorf("Path should be /custom/path, got %q", result.Path)
		}
		if len(result.DefaultRecipients) != 2 {
			t.Errorf("Expected 2 recipients, got %d", len(result.DefaultRecipients))
		}
	})

	t.Run("empty path does not override", func(t *testing.T) {
		localDefaults := VaultConfig{Path: "/default/path"}
		fileCfg := &fileVaultConfig{
			Path:              "",
			DefaultRecipients: nil,
		}
		result := MergeFileVaultConfig(fileCfg, localDefaults)

		if result.Path != "/default/path" {
			t.Errorf("Path should remain /default/path, got %q", result.Path)
		}
	})

	t.Run("ConfirmRemove is merged", func(t *testing.T) {
		confirm := true
		fileCfg := &fileVaultConfig{
			ConfirmRemove: &confirm,
		}
		result := MergeFileVaultConfig(fileCfg, defaults)

		if !result.ConfirmRemove {
			t.Error("ConfirmRemove should be true")
		}
	})

	t.Run("recipients are copied not referenced", func(t *testing.T) {
		fileCfg := &fileVaultConfig{
			DefaultRecipients: []string{"recipient1"},
		}
		result := MergeFileVaultConfig(fileCfg, defaults)

		result.DefaultRecipients[0] = "modified"
		if fileCfg.DefaultRecipients[0] == "modified" {
			t.Error("Recipients should be copied, not referenced")
		}
	})
}

func TestMergeFileGitConfig(t *testing.T) {
	defaults := defaultGitConfig()

	t.Run("nil file config returns defaults", func(t *testing.T) {
		result := MergeFileGitConfig(nil, defaults)
		if result.AutoPush != defaults.AutoPush {
			t.Error("AutoPush should be default")
		}
	})

	t.Run("file config overrides defaults", func(t *testing.T) {
		autoPush := false
		commitTemplate := "Custom template"
		fileCfg := &fileGitConfig{
			AutoPush:       &autoPush,
			CommitTemplate: &commitTemplate,
		}
		result := MergeFileGitConfig(fileCfg, defaults)

		if result.AutoPush {
			t.Error("AutoPush should be false")
		}
		if result.CommitTemplate != "Custom template" {
			t.Errorf("CommitTemplate mismatch: got %q", result.CommitTemplate)
		}
	})

	t.Run("nil pointers do not override", func(t *testing.T) {
		fileCfg := &fileGitConfig{
			AutoPush:       nil,
			CommitTemplate: nil,
		}
		result := MergeFileGitConfig(fileCfg, defaults)

		if !result.AutoPush {
			t.Error("AutoPush should remain true")
		}
		if result.CommitTemplate != defaults.CommitTemplate {
			t.Error("CommitTemplate should remain default")
		}
	})
}

func TestMergeFileMCPConfig(t *testing.T) {
	defaults := defaultMCPConfig()

	t.Run("nil file config returns defaults", func(t *testing.T) {
		result := MergeFileMCPConfig(nil, defaults)
		if result.Port != defaults.Port {
			t.Errorf("Port should be %d, got %d", defaults.Port, result.Port)
		}
	})

	t.Run("file config overrides defaults", func(t *testing.T) {
		port := 9090
		stdio := true
		bind := "0.0.0.0"
		tokenFile := "/custom/token"
		fileCfg := &fileMCPConfig{
			Port:          &port,
			Stdio:         &stdio,
			Bind:          &bind,
			HTTPTokenFile: &tokenFile,
		}
		result := MergeFileMCPConfig(fileCfg, defaults)

		if result.Port != 9090 {
			t.Errorf("Port should be 9090, got %d", result.Port)
		}
		if !result.Stdio {
			t.Error("Stdio should be true")
		}
		if result.Bind != "0.0.0.0" {
			t.Errorf("Bind should be 0.0.0.0, got %q", result.Bind)
		}
		if result.HTTPTokenFile != "/custom/token" {
			t.Errorf("HTTPTokenFile should be /custom/token, got %q", result.HTTPTokenFile)
		}
	})

	t.Run("nil pointers do not override", func(t *testing.T) {
		fileCfg := &fileMCPConfig{
			Port:  nil,
			Stdio: nil,
			Bind:  nil,
		}
		result := MergeFileMCPConfig(fileCfg, defaults)

		if result.Port != defaults.Port {
			t.Errorf("Port should remain %d, got %d", defaults.Port, result.Port)
		}
		if result.Stdio != defaults.Stdio {
			t.Error("Stdio should remain default")
		}
		if result.Bind != defaults.Bind {
			t.Errorf("Bind should remain %q, got %q", defaults.Bind, result.Bind)
		}
	})
}

func TestVaultConfigTypes(t *testing.T) {
	cfg := VaultConfig{
		Path:              "/test/path",
		DefaultRecipients: []string{"recipient1"},
	}

	if cfg.Path != "/test/path" {
		t.Error("Path mismatch")
	}
	if len(cfg.DefaultRecipients) != 1 {
		t.Error("Recipients length mismatch")
	}
}

func TestGitConfigTypes(t *testing.T) {
	cfg := GitConfig{
		AutoPush:       true,
		CommitTemplate: "Test template",
	}

	if !cfg.AutoPush {
		t.Error("AutoPush should be true")
	}
	if cfg.CommitTemplate != "Test template" {
		t.Error("CommitTemplate mismatch")
	}
}

func TestMCPConfigTypes(t *testing.T) {
	cfg := MCPConfig{
		Port:          8080,
		Bind:          "127.0.0.1",
		Stdio:         false,
		HTTPTokenFile: "auto",
	}

	if cfg.Port != 8080 {
		t.Errorf("Port should be 8080, got %d", cfg.Port)
	}
	if cfg.Bind != "127.0.0.1" {
		t.Errorf("Bind should be 127.0.0.1, got %q", cfg.Bind)
	}
	if cfg.HTTPTokenFile != "auto" {
		t.Errorf("HTTPTokenFile should be auto, got %q", cfg.HTTPTokenFile)
	}
	if cfg.Stdio {
		t.Error("Stdio should be false")
	}
}

func TestAgentProfileTypes(t *testing.T) {
	profile := AgentProfile{
		Name:         "test-agent",
		AllowedPaths: []string{"path1", "path2"},
		CanWrite:     true,
		ApprovalMode: "none",
	}

	if profile.Name != "test-agent" {
		t.Error("Name mismatch")
	}
	if len(profile.AllowedPaths) != 2 {
		t.Error("AllowedPaths length mismatch")
	}
	if !profile.CanWrite {
		t.Error("CanWrite should be true")
	}
	if profile.ApprovalMode != "none" {
		t.Errorf("ApprovalMode = %q, want none", profile.ApprovalMode)
	}
}

func TestDefaultClipboardConfig(t *testing.T) {
	cfg := defaultClipboardConfig()

	if cfg.AutoClearDuration != 30 {
		t.Errorf("AutoClearDuration should be 30, got %d", cfg.AutoClearDuration)
	}
}

func TestMergeFileClipboardConfig(t *testing.T) {
	defaults := defaultClipboardConfig()

	t.Run("nil file config returns defaults", func(t *testing.T) {
		result := MergeFileClipboardConfig(nil, defaults)
		if result.AutoClearDuration != defaults.AutoClearDuration {
			t.Errorf("AutoClearDuration should be %d, got %d", defaults.AutoClearDuration, result.AutoClearDuration)
		}
	})

	t.Run("file config overrides defaults", func(t *testing.T) {
		duration := 60
		fileCfg := &fileClipboardConfig{
			AutoClearDuration: &duration,
		}
		result := MergeFileClipboardConfig(fileCfg, defaults)

		if result.AutoClearDuration != 60 {
			t.Errorf("AutoClearDuration should be 60, got %d", result.AutoClearDuration)
		}
	})

	t.Run("nil pointer does not override", func(t *testing.T) {
		fileCfg := &fileClipboardConfig{
			AutoClearDuration: nil,
		}
		result := MergeFileClipboardConfig(fileCfg, defaults)

		if result.AutoClearDuration != defaults.AutoClearDuration {
			t.Errorf("AutoClearDuration should remain %d, got %d", defaults.AutoClearDuration, result.AutoClearDuration)
		}
	})
}
