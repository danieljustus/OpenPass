package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
)

func TestGetAutoClearDuration(t *testing.T) {
	t.Run("returns default when vaultPath fails", func(t *testing.T) {
		origVault := vault
		origFlagChanged := vaultFlag.Changed
		defer func() {
			vault = origVault
			vaultFlag.Changed = origFlagChanged
		}()

		origHome := os.Getenv("HOME")
		defer func() { _ = os.Setenv("HOME", origHome) }()
		_ = os.Unsetenv("HOME")
		_ = os.Unsetenv("OPENPASS_VAULT")

		vault = "~/.openpass"

		duration := getAutoClearDuration()
		if duration != 30 {
			t.Errorf("duration = %d, want 30", duration)
		}
	})

	t.Run("returns default when config file missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		origVault := vault
		origFlagChanged := vaultFlag.Changed
		defer func() {
			vault = origVault
			vaultFlag.Changed = origFlagChanged
		}()

		vault = tmpDir

		duration := getAutoClearDuration()
		if duration != 30 {
			t.Errorf("duration = %d, want 30", duration)
		}
	})

	t.Run("returns default when clipboard config nil", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfgPath := filepath.Join(tmpDir, "config.yaml")
		yamlContent := `mcp:
  bind: 127.0.0.1
`
		if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		origVault := vault
		origFlagChanged := vaultFlag.Changed
		defer func() {
			vault = origVault
			vaultFlag.Changed = origFlagChanged
		}()

		vault = tmpDir

		duration := getAutoClearDuration()
		if duration != 30 {
			t.Errorf("duration = %d, want 30", duration)
		}
	})

	t.Run("returns config value when set", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfgPath := filepath.Join(tmpDir, "config.yaml")
		yamlContent := `clipboard:
  auto_clear_duration: 60
`
		if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		origVault := vault
		origFlagChanged := vaultFlag.Changed
		defer func() {
			vault = origVault
			vaultFlag.Changed = origFlagChanged
		}()

		vault = tmpDir

		duration := getAutoClearDuration()
		if duration != 60 {
			t.Errorf("duration = %d, want 60", duration)
		}
	})
}

func TestGetAutoClearDurationFromConfig(t *testing.T) {
	t.Run("zero means disabled", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfgPath := filepath.Join(tmpDir, "config.yaml")
		yamlContent := `clipboard:
  auto_clear_duration: 0
`
		if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		origVault := vault
		origFlagChanged := vaultFlag.Changed
		defer func() {
			vault = origVault
			vaultFlag.Changed = origFlagChanged
		}()

		vault = tmpDir

		duration := getAutoClearDuration()
		if duration != 0 {
			t.Errorf("duration = %d, want 0", duration)
		}
	})

	t.Run("custom value", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfgPath := filepath.Join(tmpDir, "config.yaml")
		yamlContent := `clipboard:
  auto_clear_duration: 120
`
		if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		origVault := vault
		origFlagChanged := vaultFlag.Changed
		defer func() {
			vault = origVault
			vaultFlag.Changed = origFlagChanged
		}()

		vault = tmpDir

		duration := getAutoClearDuration()
		if duration != 120 {
			t.Errorf("duration = %d, want 120", duration)
		}
	})
}

func TestLoadConfigForGetAutoClearDuration(t *testing.T) {
	t.Run("load valid config with clipboard", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfgPath := filepath.Join(tmpDir, "config.yaml")
		yamlContent := `
clipboard:
  auto_clear_duration: 45
agents:
  test:
    allowedPaths: ["*"]
    canWrite: true
`
		if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if cfg.Clipboard == nil {
			t.Fatal("clipboard config is nil")
		}
		if cfg.Clipboard.AutoClearDuration != 45 {
			t.Errorf("autoClearDuration = %d, want 45", cfg.Clipboard.AutoClearDuration)
		}
	})
}
