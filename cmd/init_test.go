package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitpkg "github.com/danieljustus/OpenPass/internal/git"
)

func TestInitCommand_HiddenPassphrase(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "test-hidden-passphrase"

	restore := pipeStdin(t, passphrase+"\n")
	defer restore()

	rootCmd.SetArgs([]string{"init", vaultDir})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("init command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Vault initialized") {
		t.Errorf("expected 'Vault initialized' in output, got: %s", output)
	}

	if !strings.Contains(output, "Public key:") {
		t.Errorf("expected 'Public key:' in output, got: %s", output)
	}

	cfgPath := filepath.Join(vaultDir, "config.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Errorf("config.yaml not created at %s", cfgPath)
	}

	identityPath := filepath.Join(vaultDir, "identity.age")
	if _, err := os.Stat(identityPath); os.IsNotExist(err) {
		t.Errorf("identity.age not created at %s", identityPath)
	}

	_ = gitpkg.Init(vaultDir)
	gitDir := filepath.Join(vaultDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Errorf(".git directory not created at %s", gitDir)
	}
}
