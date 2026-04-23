package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestVersionOutputEndsWithNewline(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := buf.String()
	if len(got) == 0 {
		t.Fatal("output is empty")
	}

	if got[len(got)-1] != '\n' {
		t.Errorf("output ends with %q, want newline", got[len(got)-1])
	}

	if bytes.Contains([]byte(got), []byte{'\\', 'n'}) {
		t.Error("output contains literal backslash-n instead of newline")
	}
}

func TestVersionOutputContainsExpectedFields(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := buf.String()
	expectedPrefix := "openpass"
	if !bytes.HasPrefix([]byte(got), []byte(expectedPrefix)) {
		t.Errorf("output missing expected prefix %q, got %q", expectedPrefix, got)
	}

	if !bytes.Contains([]byte(got), []byte("(commit:")) {
		t.Errorf("output missing '(commit:' label, got %q", got)
	}

	if !bytes.Contains([]byte(got), []byte("built:")) {
		t.Errorf("output missing 'built:' label, got %q", got)
	}
}

func TestVersionCommandDoesNotRequireVault(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	originalHome := os.Getenv("HOME")
	originalVaultEnv := os.Getenv("OPENPASS_VAULT")
	originalVault := vault
	originalChanged := vaultFlag.Changed
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	vault = "~/.openpass"
	if vaultFlag != nil {
		_ = vaultFlag.Value.Set(vault)
		vaultFlag.Changed = false
	}
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
		_ = os.Setenv("OPENPASS_VAULT", originalVaultEnv)
		vault = originalVault
		if vaultFlag != nil {
			_ = vaultFlag.Value.Set(originalVault)
			vaultFlag.Changed = originalChanged
		}
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := buf.String(); !bytes.Contains([]byte(got), []byte("openpass")) {
		t.Fatalf("unexpected output: %q", got)
	}
}
