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

func TestSetCommand_InvalidTOTPSecretRejected(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "bad-totp-set",
			"--value", "pass", "--totp-secret", "not-valid-base32!!!"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})

	if !strings.Contains(stderr, "TOTP secret must be Base32-encoded") {
		t.Errorf("expected TOTP validation error, got: %s", stderr)
	}
	if strings.Contains(stderr, "not-valid-base32!!!") {
		t.Error("error message must not contain the secret value")
	}
}

func TestSetCommand_ValidTOTPSecretAccepted(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	out := execWithStdout("--vault", vaultDir, "set", "valid-totp-set",
		"--value", "pass", "--totp-secret", "JBSWY3DPEHPK3PXP")
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
}

func TestSetCommand_TOTPSecretWithSpacesAccepted(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	out := execWithStdout("--vault", vaultDir, "set", "spaced-totp-set",
		"--value", "pass", "--totp-secret", "JBSW Y3DP EHPK 3PXP")
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
}
