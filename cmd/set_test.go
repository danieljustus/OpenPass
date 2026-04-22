package cmd

import (
	"strings"
	"testing"
)

func TestSetCommand_HiddenPassword(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	restore := pipeStdin(t, "myuser\nnew-secret-password\n\n\n")
	defer restore()

	rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "test-entry"})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("set command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Entry saved") {
		t.Errorf("expected 'Entry saved' in output, got: %s", output)
	}
}
