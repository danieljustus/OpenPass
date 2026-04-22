package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
	gitpkg "github.com/danieljustus/OpenPass/internal/git"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func resetCmdFlags() {
	resetCommandTestState()
}

func setupVaultFlag(t *testing.T, vaultDir string) func() {
	t.Helper()
	origVault := vault
	origChanged := vaultFlag.Changed
	_ = vaultFlag.Value.Set(vaultDir)
	vaultFlag.Changed = true
	vault = vaultDir
	return func() {
		vault = origVault
		_ = vaultFlag.Value.Set(origVault)
		vaultFlag.Changed = origChanged
	}
}

func initVault(t *testing.T) (string, string) {
	t.Helper()
	resetCmdFlags()
	t.Cleanup(resetCmdFlags)
	vaultDir := t.TempDir()
	passphrase := "test-passphrase"
	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}
	return vaultDir, passphrase
}

func setPassEnv(t *testing.T, passphrase string) {
	t.Helper()
	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })
}

func fakeEditorWithContent(t *testing.T, content string) string {
	t.Helper()
	contentFile := filepath.Join(t.TempDir(), "edited.json")
	if err := os.WriteFile(contentFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write content file: %v", err)
	}
	editorFile := filepath.Join(t.TempDir(), "fake_editor")
	script := fmt.Sprintf("#!/bin/sh\ncp '%s' \"$1\"\n", contentFile)
	if err := os.WriteFile(editorFile, []byte(script), 0o755); err != nil {
		t.Fatalf("write editor: %v", err)
	}
	return editorFile
}

func fakeEditorEmpty(t *testing.T) string {
	t.Helper()
	editorFile := filepath.Join(t.TempDir(), "empty_editor")
	script := "#!/bin/sh\nprintf '' > \"$1\"\n"
	if err := os.WriteFile(editorFile, []byte(script), 0o755); err != nil {
		t.Fatalf("write editor: %v", err)
	}
	return editorFile
}

func fakeEditorInvalid(t *testing.T) string {
	t.Helper()
	editorFile := filepath.Join(t.TempDir(), "invalid_editor")
	script := "#!/bin/sh\nprintf '{not valid json}' > \"$1\"\n"
	if err := os.WriteFile(editorFile, []byte(script), 0o755); err != nil {
		t.Fatalf("write editor: %v", err)
	}
	return editorFile
}

func pipeStdin(t *testing.T, input string) func() {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	_, _ = w.WriteString(input)
	_ = w.Close()
	return func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}
}

func TestCmdAdd_Interactive(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "myuser\nsecretpass\n")
	defer restore()
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "interactive-entry",
			"--url", "https://example.com", "--notes", "some notes", "--totp-secret", "JBSWY3DPEHPK3PXP"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(output, "Entry created") {
		t.Errorf("expected Entry created, got: %s", output)
	}
}

func TestCmdAdd_InteractiveStdinError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "myuser\n")
	defer restore()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "add-stdin-err"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "read password") {
		t.Errorf("expected read password error, got: %s", stderr)
	}
}

func TestCmdAdd_Generate(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "add", "gen-entry", "--generate")
	if !strings.Contains(out, "Entry created") {
		t.Errorf("expected Entry created, got: %s", out)
	}
}

func TestCmdAdd_WithUsernameAndURL(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "add", "tagged-entry",
		"--value", "secret", "--username", "alice", "--url", "https://example.com")
	if !strings.Contains(out, "Entry created") {
		t.Errorf("expected Entry created, got: %s", out)
	}
}

func TestCmdAdd_WithTOTPFlags(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "add", "totp-entry",
		"--value", "pass", "--totp-secret", "JBSWY3DPEHPK3PXP", "--totp-issuer", "GitHub", "--totp-account", "alice")
	if !strings.Contains(out, "Entry created") {
		t.Errorf("expected Entry created, got: %s", out)
	}
}

func TestCmdAdd_Uninitialized(t *testing.T) {
	resetCmdFlags()
	t.Cleanup(resetCmdFlags)
	vaultDir := t.TempDir()
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "x", "--value", "v"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "vault not initialized") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected vault not initialized error, got: %s", stderr)
	}
}

func TestCmdAdd_AlreadyExists(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "existing"}}
	_ = vaultpkg.WriteEntry(vaultDir, "exists", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "exists", "--value", "v"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "already exists") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected already exists error, got: %s", stderr)
	}
}

func TestCmdEdit_Success(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "original"}}
	_ = vaultpkg.WriteEntry(vaultDir, "edit-me", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	editor := fakeEditorWithContent(t,
		`{"data":{"password":"edited_pass"},"meta":{"created":"0001-01-01T00:00:00Z","updated":"0001-01-01T00:00:00Z","version":0}}`)
	origEditor := os.Getenv("EDITOR")
	_ = os.Setenv("EDITOR", editor)
	defer func() { _ = os.Setenv("EDITOR", origEditor) }()
	out := execWithStdout("--vault", vaultDir, "edit", "edit-me")
	if !strings.Contains(out, "Entry updated") {
		t.Errorf("expected Entry updated, got: %s", out)
	}
}

func TestCmdEdit_NotFound(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "edit", "ghost"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected not found, got: %s", stderr)
	}
}

func TestCmdEdit_EditorNotFound(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "x"}}
	_ = vaultpkg.WriteEntry(vaultDir, "ed-nf", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	origEditor := os.Getenv("EDITOR")
	_ = os.Setenv("EDITOR", "nonexistent_editor_xyz_abc")
	defer func() { _ = os.Setenv("EDITOR", origEditor) }()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "edit", "ed-nf"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected editor not found error, got: %s", stderr)
	}
}

//nolint:dupl // test coverage helper with similar structure to invalid JSON test
func TestCmdEdit_EmptyFile(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "x"}}
	_ = vaultpkg.WriteEntry(vaultDir, "empty-edit", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	origEditor := os.Getenv("EDITOR")
	_ = os.Setenv("EDITOR", fakeEditorEmpty(t))
	defer func() { _ = os.Setenv("EDITOR", origEditor) }()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "edit", "empty-edit"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "empty") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected empty file error, got: %s", stderr)
	}
}

//nolint:dupl // test coverage helper with similar structure to empty file test
func TestCmdEdit_InvalidJSON(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "x"}}
	_ = vaultpkg.WriteEntry(vaultDir, "bad-json", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	origEditor := os.Getenv("EDITOR")
	_ = os.Setenv("EDITOR", fakeEditorInvalid(t))
	defer func() { _ = os.Setenv("EDITOR", origEditor) }()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "edit", "bad-json"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "invalid JSON") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected invalid JSON error, got: %s", stderr)
	}
}

func TestCmdGet_WholeEntry(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "mypass", "username": "bob"}}
	_ = vaultpkg.WriteEntry(vaultDir, "full-entry", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "get", "full-entry")
	if !strings.Contains(out, "full-entry") || !strings.Contains(out, "mypass") {
		t.Errorf("expected full entry output, got: %s", out)
	}
}

func TestCmdGet_FieldNotFound(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "mypass"}}
	_ = vaultpkg.WriteEntry(vaultDir, "getfield", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "get", "getfield.nofield"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "field not found") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected field not found, got: %s", stderr)
	}
}

func TestCmdGet_NotFound(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "get", "ghost-entry"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected entry not found, got: %s", stderr)
	}
}

func TestCmdGet_FuzzyMultipleMatches(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	e := &vaultpkg.Entry{Data: map[string]any{"password": "p"}}
	_ = vaultpkg.WriteEntry(vaultDir, "work/aws", e, identity.Identity)
	_ = vaultpkg.WriteEntry(vaultDir, "work/gcp", e, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	allOutput := captureStdout(func() {
		captureStderr(func() {
			rootCmd.SetArgs([]string{"--vault", vaultDir, "get", "work"})
			_ = rootCmd.Execute()
			rootCmd.SetArgs(nil)
		})
	})
	_ = allOutput
}

func TestCmdGet_TOTP(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{
		Data: map[string]any{
			"password": "pass",
			"totp": map[string]any{
				"secret": "JBSWY3DPEHPK3PXP",
			},
		},
	}
	_ = vaultpkg.WriteEntry(vaultDir, "totp-get", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "get", "totp-get")
	if !strings.Contains(out, "totp-get") {
		t.Errorf("expected entry path in output, got: %s", out)
	}
}

func TestCmdSet_FieldSyntax(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "set", "myapp.username", "--value", "alice")
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
}

func TestCmdSet_MergeExisting(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "old"}}
	_ = vaultpkg.WriteEntry(vaultDir, "merge-me", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "set", "merge-me", "--value", "updated")
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
}

func TestCmdSet_TOTP(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "set", "totp-set", "--value", "pass",
		"--totp-secret", "JBSWY3DPEHPK3PXP", "--totp-issuer", "GitHub")
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
}

func TestCmdSet_InteractiveField(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "newvalue\n")
	defer restore()
	out := captureStdout(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "ifield.custom"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
}

func TestCmdSet_InteractiveFieldStdinError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "")
	defer restore()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "ifield.custom"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "read value") {
		t.Errorf("expected read value error, got: %s", stderr)
	}
}

func TestCmdSet_Interactive(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "alice\nmypassword\nhttps://example.com\n\n")
	defer restore()
	out := captureStdout(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "interactive-set"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
}

func TestCmdSet_InteractiveStdinError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "")
	defer restore()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "stdin-err"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "read username") {
		t.Errorf("expected read username error, got: %s", stderr)
	}
}

func TestCmdDelete_Cancel(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "keep"}}
	_ = vaultpkg.WriteEntry(vaultDir, "keep-me", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("n\n")
	_ = w.Close()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "delete", "keep-me"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	os.Stdin = oldStdin
	_ = r.Close()
	if !strings.Contains(stderr, "Canceled") {
		t.Errorf("expected Canceled, got: %s", stderr)
	}
}

func TestCmdDelete_StdinError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "keep"}}
	_ = vaultpkg.WriteEntry(vaultDir, "keep-me", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "")
	defer restore()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "delete", "keep-me"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "read confirmation") {
		t.Errorf("expected read confirmation error, got: %s", stderr)
	}
}

func TestCmdDelete_NotFound(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("y\n")
	_ = w.Close()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "delete", "ghost"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	os.Stdin = oldStdin
	_ = r.Close()
	if !strings.Contains(stderr, "Error") && !strings.Contains(stderr, "cannot delete") {
		t.Errorf("expected delete error, got: %s", stderr)
	}
}

func TestCmdDelete_YesJSON(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "secret"}}
	_ = vaultpkg.WriteEntry(vaultDir, "delete-json", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	t.Cleanup(func() {
		deleteYes = false
		deleteJSON = false
	})

	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "delete", "delete-json", "--yes", "--json"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("delete --json output is not valid JSON: %q: %v", output, err)
	}
	if result["deleted"] != true || result["path"] != "delete-json" {
		t.Fatalf("unexpected delete JSON result: %#v", result)
	}
}

func TestCmdFind_NoMatches(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "find", "nomatch_xyz_abc"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "No matches") {
		t.Errorf("expected No matches, got: %s", stderr)
	}
}

func TestCmdFind_WithFieldMatches(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "uniquevalue123"}}
	_ = vaultpkg.WriteEntry(vaultDir, "find-me", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "find", "find-me")
	if !strings.Contains(out, "find-me") {
		t.Errorf("expected find-me in output, got: %s", out)
	}
}

func TestCmdList_Empty(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "list")
	_ = out
}

func TestCmdList_Prefix(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	e := &vaultpkg.Entry{Data: map[string]any{"password": "p"}}
	_ = vaultpkg.WriteEntry(vaultDir, "work/aws", e, identity.Identity)
	_ = vaultpkg.WriteEntry(vaultDir, "personal/bank", e, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "list", "work/")
	if !strings.Contains(out, "work/aws") {
		t.Errorf("expected work/aws in output, got: %s", out)
	}
	if strings.Contains(out, "personal") {
		t.Errorf("unexpected personal in prefix-filtered output: %s", out)
	}
}

func TestCmdList_Alias(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "p"}}
	_ = vaultpkg.WriteEntry(vaultDir, "ls-entry", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "ls")
	if !strings.Contains(out, "ls-entry") {
		t.Errorf("expected ls-entry in output, got: %s", out)
	}
}

func TestCmdFind_SearchAlias(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "searchval"}}
	_ = vaultpkg.WriteEntry(vaultDir, "search-me", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "search", "search-me")
	if !strings.Contains(out, "search-me") {
		t.Errorf("expected search-me in output, got: %s", out)
	}
}

func TestCmdAdd_Notes(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "add", "notes-entry",
		"--value", "pass", "--notes", "important note here")
	if !strings.Contains(out, "Entry created") {
		t.Errorf("expected Entry created, got: %s", out)
	}
}

func TestCmdSet_Uninitialized(t *testing.T) {
	resetCmdFlags()
	t.Cleanup(resetCmdFlags)
	vaultDir := t.TempDir()
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "x", "--value", "v"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "vault not initialized") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected vault not initialized, got: %s", stderr)
	}
}

func TestCmdGet_Uninitialized(t *testing.T) {
	resetCmdFlags()
	t.Cleanup(resetCmdFlags)
	vaultDir := t.TempDir()
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "get", "x"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "vault not initialized") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected vault not initialized, got: %s", stderr)
	}
}

func TestCmdList_Uninitialized(t *testing.T) {
	resetCmdFlags()
	t.Cleanup(resetCmdFlags)
	vaultDir := t.TempDir()
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "list"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "vault not initialized") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected vault not initialized, got: %s", stderr)
	}
}

func TestCmdFind_Uninitialized(t *testing.T) {
	resetCmdFlags()
	t.Cleanup(resetCmdFlags)
	vaultDir := t.TempDir()
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "find", "x"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "vault not initialized") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected vault not initialized, got: %s", stderr)
	}
}

func TestCmdDelete_Uninitialized(t *testing.T) {
	resetCmdFlags()
	t.Cleanup(resetCmdFlags)
	vaultDir := t.TempDir()
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "y\n")
	defer restore()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "delete", "x"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "vault not initialized") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected vault not initialized, got: %s", stderr)
	}
}

func TestCmdEdit_Uninitialized(t *testing.T) {
	resetCmdFlags()
	t.Cleanup(resetCmdFlags)
	vaultDir := t.TempDir()
	defer setupVaultFlag(t, vaultDir)()
	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "edit", "x"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "vault not initialized") && !strings.Contains(stderr, "Error") {
		t.Errorf("expected vault not initialized, got: %s", stderr)
	}
}

func TestCmdGet_FuzzySingleMatch(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	e := &vaultpkg.Entry{Data: map[string]any{"password": "p"}}
	_ = vaultpkg.WriteEntry(vaultDir, "workstation", e, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "get", "workstat")
	if !strings.Contains(out, "workstation") {
		t.Errorf("expected fuzzy match output, got: %s", out)
	}
}

func TestCmdDelete_EmptyConfirm(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "x"}}
	_ = vaultpkg.WriteEntry(vaultDir, "del-empty", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_ = w.Close()
	captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "delete", "del-empty"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	os.Stdin = oldStdin
	_ = r.Close()
}

func TestCmdMCPConfig_HTTPMode(t *testing.T) {
	vaultDir, _ := initVault(t)
	defer setupVaultFlag(t, vaultDir)()
	cfgPath := filepath.Join(vaultDir, "config.yaml")
	cfg := "mcp:\n  bind: 127.0.0.1\n  port: 9999\n  http_token_file: \"\"\n"
	_ = os.WriteFile(cfgPath, []byte(cfg), 0o600)
	_ = execWithStdout("--vault", vaultDir, "mcp-config", "default", "--http")
}

func TestCmdAdd_InteractiveFull(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "myuser\nmypass\n")
	defer restore()
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "full-interactive",
			"--url", "https://example.com", "--notes", "important notes",
			"--totp-secret", "JBSWY3DPEHPK3PXP", "--totp-issuer", "GitHub", "--totp-account", "alice"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(output, "Entry created") {
		t.Errorf("expected Entry created, got: %s", output)
	}
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry, err := vaultpkg.ReadEntry(vaultDir, "full-interactive", identity.Identity)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if entry.Data["username"] != "myuser" {
		t.Errorf("expected username myuser, got: %v", entry.Data["username"])
	}
	if entry.Data["password"] != "mypass" {
		t.Errorf("expected password mypass, got: %v", entry.Data["password"])
	}
	if entry.Data["url"] != "https://example.com" {
		t.Errorf("expected url https://example.com, got: %v", entry.Data["url"])
	}
	if entry.Data["notes"] != "important notes" {
		t.Errorf("expected notes, got: %v", entry.Data["notes"])
	}
	totp, ok := entry.Data["totp"].(map[string]any)
	if !ok {
		t.Fatal("expected totp data in entry")
	}
	if totp["secret"] != "JBSWY3DPEHPK3PXP" {
		t.Errorf("expected totp secret JBSWY3DPEHPK3PXP, got: %v", totp["secret"])
	}
	if totp["issuer"] != "GitHub" {
		t.Errorf("expected totp issuer GitHub, got: %v", totp["issuer"])
	}
	if totp["account_name"] != "alice" {
		t.Errorf("expected totp account_name alice, got: %v", totp["account_name"])
	}
}

func TestCmdAdd_InteractiveURLPrompt(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "testuser\ntestpass\n")
	defer restore()
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "url-prompt-entry",
			"--url", "https://mysite.com", "--totp-secret", "JBSWY3DPEHPK3PXP"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(output, "Entry created") {
		t.Errorf("expected Entry created, got: %s", output)
	}
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry, err := vaultpkg.ReadEntry(vaultDir, "url-prompt-entry", identity.Identity)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if entry.Data["url"] != "https://mysite.com" {
		t.Errorf("expected url https://mysite.com, got: %v", entry.Data["url"])
	}
	if entry.Data["username"] != "testuser" {
		t.Errorf("expected username testuser, got: %v", entry.Data["username"])
	}
}

func TestCmdGet_TOTPDisplay(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{
		Data: map[string]any{
			"password": "pass",
			"totp": map[string]any{
				"secret": "JBSWY3DPEHPK3PXP",
			},
		},
	}
	_ = vaultpkg.WriteEntry(vaultDir, "totp-display", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "get", "totp-display")
	if !strings.Contains(out, "TOTP Code") {
		t.Errorf("expected TOTP Code in output, got: %s", out)
	}
	if !strings.Contains(out, "expires in") {
		t.Errorf("expected 'expires in' in output, got: %s", out)
	}
}

func TestCmdGet_FieldWithClipboard(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte("clipboard:\n  auto_clear_duration: 0\n"), 0o600); err != nil {
		t.Fatalf("disable clipboard auto-clear: %v", err)
	}
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "clip-pass-123"}}
	_ = vaultpkg.WriteEntry(vaultDir, "clip-entry", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	var copied string
	origClipboardWriteAll := getClipboardWriteAll
	getClipboardWriteAll = func(s string) error {
		copied = s
		return nil
	}
	t.Cleanup(func() { getClipboardWriteAll = origClipboardWriteAll })

	var stdout string
	var execErr error
	stderr := captureStderr(func() {
		stdout = captureStdout(func() {
			rootCmd.SetArgs([]string{"--vault", vaultDir, "get", "clip-entry.password", "--clip"})
			execErr = rootCmd.Execute()
			rootCmd.SetArgs(nil)
		})
	})
	if execErr != nil {
		t.Fatalf("get command failed: %v", execErr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty when --clip is set", stdout)
	}
	if copied != "clip-pass-123" {
		t.Fatalf("copied = %q, want clip-pass-123", copied)
	}
	if !strings.Contains(stderr, "[copied to clipboard]") {
		t.Fatalf("stderr = %q, want copied status", stderr)
	}
}

func TestCmdSet_InteractiveFull(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "alice\nmypassword\nhttps://example.com\n\n")
	defer restore()
	out := captureStdout(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "interactive-full"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry, err := vaultpkg.ReadEntry(vaultDir, "interactive-full", identity.Identity)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if entry.Data["username"] != "alice" {
		t.Errorf("expected username alice, got: %v", entry.Data["username"])
	}
	if entry.Data["password"] != "mypassword" {
		t.Errorf("expected password mypassword, got: %v", entry.Data["password"])
	}
	if entry.Data["url"] != "https://example.com" {
		t.Errorf("expected url https://example.com, got: %v", entry.Data["url"])
	}
}

func TestCmdSet_InteractiveFieldWithTOTP(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	restore := pipeStdin(t, "fieldvalue\n")
	defer restore()
	out := captureStdout(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "totpfield.custom",
			"--totp-secret", "JBSWY3DPEHPK3PXP", "--totp-issuer", "MyService"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(out, "Entry saved") {
		t.Errorf("expected Entry saved, got: %s", out)
	}
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry, err := vaultpkg.ReadEntry(vaultDir, "totpfield", identity.Identity)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if entry.Data["custom"] != "fieldvalue" {
		t.Errorf("expected custom fieldvalue, got: %v", entry.Data["custom"])
	}
	totp, ok := entry.Data["totp"].(map[string]any)
	if !ok {
		t.Fatal("expected totp data in entry")
	}
	if totp["secret"] != "JBSWY3DPEHPK3PXP" {
		t.Errorf("expected totp secret, got: %v", totp["secret"])
	}
}

func TestCmdGet_FuzzyFieldLookup(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "secret123", "username": "admin"}}
	_ = vaultpkg.WriteEntry(vaultDir, "work/aws", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	out := execWithStdout("--vault", vaultDir, "get", "work/aws.password")
	if !strings.Contains(out, "secret123") {
		t.Errorf("expected secret123 in output: %s", out)
	}
}

func TestCmdEdit_EditorRunError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "x"}}
	_ = vaultpkg.WriteEntry(vaultDir, "edit-err", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	origEditor := os.Getenv("EDITOR")
	_ = os.Setenv("EDITOR", "false")
	defer func() { _ = os.Setenv("EDITOR", origEditor) }()

	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "edit", "edit-err"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "editor failed") {
		t.Errorf("expected 'editor failed' in stderr: %s", stderr)
	}
}

func TestCmdGet_TOTPGenerationError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{
		Data: map[string]any{
			"password": "pass",
			"totp": map[string]any{
				"secret": "INVALID!!!SECRET!!!",
			},
		},
	}
	_ = vaultpkg.WriteEntry(vaultDir, "bad-totp", entry, identity.Identity)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	stderr := captureStderr(func() {
		_ = execWithStdout("--vault", vaultDir, "get", "bad-totp")
	})
	if !strings.Contains(stderr, "Warning: could not generate TOTP code") {
		t.Errorf("expected TOTP warning in stderr, got: %s", stderr)
	}
}

func setupGitWithBrokenObjects(t *testing.T, vaultDir string) {
	t.Helper()
	if err := gitpkg.Init(vaultDir); err != nil {
		t.Fatalf("init git: %v", err)
	}
	objectsDir := filepath.Join(vaultDir, ".git", "objects")
	if err := os.Chmod(objectsDir, 0o000); err != nil {
		t.Fatalf("chmod objects dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(objectsDir, 0o700)
	})
}

func TestCmdSet_AutoCommitError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setupGitWithBrokenObjects(t, vaultDir)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "set", "autocommit-set", "--value", "secret"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "auto-commit failed") {
		t.Errorf("expected auto-commit warning in stderr: %s", stderr)
	}
}

func TestCmdAdd_AutoCommitError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setupGitWithBrokenObjects(t, vaultDir)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "add", "autocommit-add", "--value", "secret"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	if !strings.Contains(stderr, "auto-commit failed") {
		t.Errorf("expected auto-commit warning in stderr: %s", stderr)
	}
}

func TestCmdDelete_AutoCommitError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "del"}}
	_ = vaultpkg.WriteEntry(vaultDir, "autocommit-del", entry, identity.Identity)
	setupGitWithBrokenObjects(t, vaultDir)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("y\n")
	_ = w.Close()

	stderr := captureStderr(func() {
		rootCmd.SetArgs([]string{"--vault", vaultDir, "delete", "autocommit-del"})
		_ = rootCmd.Execute()
		rootCmd.SetArgs(nil)
	})
	os.Stdin = oldStdin
	_ = r.Close()

	if !strings.Contains(stderr, "auto-commit failed") {
		t.Errorf("expected auto-commit warning in stderr: %s", stderr)
	}
}

func TestCmdEdit_AutoCommitError(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	identity, _ := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	entry := &vaultpkg.Entry{Data: map[string]any{"password": "original"}}
	_ = vaultpkg.WriteEntry(vaultDir, "autocommit-edit", entry, identity.Identity)
	setupGitWithBrokenObjects(t, vaultDir)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	editor := fakeEditorWithContent(t,
		`{"data":{"password":"edited"},"meta":{"created":"0001-01-01T00:00:00Z","updated":"0001-01-01T00:00:00Z","version":0}}`)
	origEditor := os.Getenv("EDITOR")
	_ = os.Setenv("EDITOR", editor)
	defer func() { _ = os.Setenv("EDITOR", origEditor) }()

	stderr := captureStderr(func() {
		_ = execWithStdout("--vault", vaultDir, "edit", "autocommit-edit")
	})
	if !strings.Contains(stderr, "auto-commit failed") {
		t.Errorf("expected auto-commit warning in stderr: %s", stderr)
	}
}
