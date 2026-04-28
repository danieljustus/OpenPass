package cmd

import (
	"strings"
	"testing"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func TestAddCommand_HiddenPassword(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	restore := pipeStdin(t, "myuser\nsecret-password\n\n\n\n")
	defer restore()

	rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "test-entry"})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Entry created") {
		t.Errorf("expected 'Entry created' in output, got: %s", output)
	}
}

func TestAddCommand_InvalidTOTPSecretRejected(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "bad-totp-entry",
			"--value", "StrongP@ssw0rd123", "--totp-secret", "not-valid-base32!!!"})
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

func TestAddCommand_ValidTOTPSecretAccepted(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	out := execWithStdout("--vault", vaultDir, "add", "valid-totp-entry",
		"--value", "StrongP@ssw0rd123", "--totp-secret", "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ")
	if !strings.Contains(out, "Entry created") {
		t.Errorf("expected Entry created, got: %s", out)
	}
}

func TestAddCommand_TOTPSecretWithSpacesAccepted(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	out := execWithStdout("--vault", vaultDir, "add", "spaced-totp-entry",
		"--value", "StrongP@ssw0rd123", "--totp-secret", "GEZD GNBV GY3T QOJQ GEZD GNBV GY3T QOJQ")
	if !strings.Contains(out, "Entry created") {
		t.Errorf("expected Entry created, got: %s", out)
	}
}

func TestAddCommand_GenerateWithLength(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "generated-entry", "--generate", "--length", "24"})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}
	})
	if !strings.Contains(output, "Entry created") {
		t.Errorf("expected 'Entry created' in output, got: %s", output)
	}

	v, err := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	entry, err := vaultpkg.ReadEntry(vaultDir, "generated-entry", v.Identity)
	if err != nil {
		t.Fatalf("read generated entry: %v", err)
	}
	password, ok := entry.Data["password"].(string)
	if !ok {
		t.Fatalf("password has unexpected type %T", entry.Data["password"])
	}
	if len(password) != 24 {
		t.Fatalf("password length = %d, want 24", len(password))
	}
}
