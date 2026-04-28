package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/git"
	"github.com/danieljustus/OpenPass/internal/session"
	"github.com/danieljustus/OpenPass/internal/testutil"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func vaultFlagReset(t *testing.T) {
	t.Helper()
	origVault := vault
	origChanged := false
	if vaultFlag != nil {
		origChanged = vaultFlag.Changed
	}
	t.Cleanup(func() {
		vault = origVault
		if vaultFlag != nil {
			_ = vaultFlag.Value.Set(origVault)
			vaultFlag.Changed = origChanged
		}
	})
}

func TestCmdInit_Success(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("supersecretpassphrase123\n")
	_ = w.Close()
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})

	rootCmd.SetArgs([]string{"--vault", vaultDir, "init"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "Vault initialized") {
		t.Errorf("init output missing success message: %q", output)
	}
}

func TestCmdInit_AlreadyInitialized(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, "supersecretpassphrase123", config.Default()); err != nil {
		t.Fatalf("pre-init vault: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "init"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for already initialized vault")
	}
	if !strings.Contains(execErr.Error(), "already initialized") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdInit_ShortPassphrase(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("short\n")
	_ = w.Close()
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})

	rootCmd.SetArgs([]string{"--vault", vaultDir, "init"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for short passphrase")
	}
}

func TestCmdGitPush_NoRemote(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_VAULT") })

	if err := git.Init(vaultDir); err != nil {
		t.Fatalf("git init: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "git", "push"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "Pushed") {
		t.Logf("push output: %q (no remote → skipped, still ok)", output)
	}
}

func TestCmdGitPull_NoRemote(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_VAULT") })

	if err := git.Init(vaultDir); err != nil {
		t.Fatalf("git init: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "git", "pull"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "Pulled") {
		t.Logf("pull output: %q (no remote → skipped, still ok)", output)
	}
}

func TestCmdGitLog_Success(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}
	if err := git.Init(vaultDir); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := git.CreateGitignore(vaultDir); err != nil {
		t.Fatalf("git createGitignore: %v", err)
	}
	if err := os.WriteFile(vaultDir+"/dummy.txt", []byte("test"), 0o644); err != nil {
		t.Fatalf("write dummy: %v", err)
	}
	if err := git.AutoCommit(vaultDir, "initial commit"); err != nil {
		t.Fatalf("auto commit: %v", err)
	}

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "git", "log"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if len(strings.TrimSpace(output)) == 0 {
		t.Errorf("git log output is empty, expected commit history")
	}
}

func TestCmdGitUnknownAction(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)
	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_VAULT") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "git", "unknown"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for unknown git action")
	}
	if !strings.Contains(execErr.Error(), "unknown action") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdLock_Success(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	if err := session.SavePassphrase(vaultDir, passphrase, time.Hour); err != nil {
		t.Skipf("keyring unavailable: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "lock"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStderr(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "Vault locked") {
		t.Errorf("lock output missing expected message: %q", output)
	}
}

func TestCmdUnlock_CheckExpired(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	t.Cleanup(func() { _ = unlockCmd.Flags().Set("check", "false") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "unlock", "--check"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for expired/missing session")
	}
	if !strings.Contains(execErr.Error(), "no active session") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdUnlock_CheckActive(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	if err := session.SavePassphrase(vaultDir, passphrase, time.Hour); err != nil {
		t.Skipf("keyring unavailable: %v", err)
	}

	t.Cleanup(func() {
		_ = unlockCmd.Flags().Set("check", "false")
		_ = session.ClearSession(vaultDir)
	})

	rootCmd.SetArgs([]string{"--vault", vaultDir, "unlock", "--check"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStderr(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "Session active") {
		t.Errorf("unlock --check output: %q", output)
	}
}

func TestCmdRecipientsList_WithRecipients(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_VAULT") })

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("add recipient: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "list"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "Recipients") {
		t.Errorf("list output missing header: %q", output)
	}
	if !strings.Contains(output, testRecipient1) {
		t.Errorf("list output missing recipient: %q", output)
	}
}

func TestCmdRecipientsAdd_Invalid(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "add", "not-a-valid-key"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for invalid recipient key")
	}
	if !strings.Contains(execErr.Error(), "invalid recipient") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdRecipientsAdd_Duplicate(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("pre-add recipient: %v", err)
	}

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "add", testRecipient1})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for duplicate recipient")
	}
	if !strings.Contains(execErr.Error(), "already exists") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdRecipientsRemove_NotFound(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "remove", testRecipient2, "--yes"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for recipient not found")
	}
	if !strings.Contains(execErr.Error(), "not found") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdRecipientsRemove_Cancel(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	origConfirmRemove := confirmRemove
	confirmRemove = false
	t.Cleanup(func() { confirmRemove = origConfirmRemove })

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("n\n")
	_ = w.Close()
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "remove", testRecipient1})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	output := captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr != nil {
		t.Errorf("unexpected error for canceled remove: %v", execErr)
	}
	if !strings.Contains(output, "Canceled") {
		t.Errorf("expected 'Canceled' in output: %q", output)
	}
}

func TestCmdGenerate_StoreExisting(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	identity, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default())
	if err != nil {
		t.Fatalf("init vault: %v", err)
	}

	entry := &vaultpkg.Entry{Data: map[string]any{"password": "oldpass"}}
	if err := vaultpkg.WriteEntry(vaultDir, "existing.password", entry, identity); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	origGenStore := genStore
	origGenLength := genLength
	t.Cleanup(func() { genStore = origGenStore; genLength = origGenLength })

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "generate", "--length", "16", "--store", "existing.password"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "Password stored at") {
		t.Errorf("generate store existing output: %q", output)
	}
}

func TestCmdGenerate_StoreJSONDoesNotRevealByDefault(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	t.Cleanup(func() {
		genStore = ""
		genJSON = false
		genReveal = false
		genQuiet = false
	})

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "generate", "--length", "16", "--store", "json.password", "--json"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("generate --store --json output is not valid JSON: %q: %v", output, err)
	}
	if _, ok := result["password"]; ok {
		t.Fatalf("generate --store --json revealed password by default: %#v", result)
	}
	if result["stored"] != true || result["path"] != "json.password" {
		t.Fatalf("unexpected generate JSON result: %#v", result)
	}
}

func TestCmdGenerate_ZeroLength(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	origGenLength := genLength
	t.Cleanup(func() { genLength = origGenLength })

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "generate", "--length", "0"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for zero length password")
	}
}

func TestCmdGenerate_NoStore(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, "testpassphrase", config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	origGenStore := genStore
	origGenLength := genLength
	t.Cleanup(func() { genStore = origGenStore; genLength = origGenLength })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "generate", "--length", "12"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if len(strings.TrimSpace(output)) != 12 {
		t.Errorf("generate output length %d, want 12: %q", len(strings.TrimSpace(output)), output)
	}
}

func TestCmdServe_EmptyBind(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_VAULT") })

	t.Cleanup(func() {
		_ = serveCmd.Flags().Set("bind", "127.0.0.1")
		_ = serveCmd.Flags().Set("stdio", "false")
	})

	rootCmd.SetArgs([]string{"--vault", vaultDir, "serve", "--bind", ""})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for empty bind address")
	}
	if !strings.Contains(execErr.Error(), "bind") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdServe_MissingAgentInStdioMode(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_VAULT") })

	t.Cleanup(func() {
		_ = serveCmd.Flags().Set("bind", "127.0.0.1")
		_ = serveCmd.Flags().Set("stdio", "false")
		_ = serveCmd.Flags().Set("agent", "")
	})
	_ = serveCmd.Flags().Set("agent", "")

	rootCmd.SetArgs([]string{"--vault", vaultDir, "serve", "--bind", "127.0.0.1", "--stdio"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for missing --agent in stdio mode")
	}
	if !strings.Contains(execErr.Error(), "--agent") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

// TestServe_RunE_StdioWithAgent is skipped because stdio mode requires
// proper JSON-RPC message handling which is difficult to test in isolation.
// The runStdioServer function itself is tested via integration tests.
// TODO(#42): Implement stdio transport unit test with mock JSON-RPC stdin/stdout.
func TestServe_RunE_StdioWithAgent(t *testing.T) {
	t.Skip("TODO(#42): stdio transport requires mock JSON-RPC stdin/stdout — covered by integration tests")
}

func TestServe_RunE_HTTPWithAgent(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}
	vaultFlagReset(t)

	port := findFreePort(t)

	serveSignals := make(chan chan<- os.Signal, 1)
	origNotify := serveSignalNotify
	serveSignalNotify = func(c chan<- os.Signal, sigs ...os.Signal) {
		serveSignals <- c
	}
	t.Cleanup(func() { serveSignalNotify = origNotify })

	origUnlock := serveUnlockVault
	serveUnlockVault = func(vaultDir string, interactive bool) (*vaultpkg.Vault, error) {
		if !interactive {
			t.Error("HTTP mode should request interactive unlock")
		}
		return &vaultpkg.Vault{Dir: vaultDir, Identity: identity, Config: cfg}, nil
	}
	t.Cleanup(func() { serveUnlockVault = origUnlock })

	started := make(chan struct{})
	origHTTP := runHTTPServerFunc
	runHTTPServerFunc = func(ctx context.Context, bind string, gotPort int, v *vaultpkg.Vault) error {
		if bind != "127.0.0.1" {
			t.Errorf("bind = %q, want 127.0.0.1", bind)
		}
		if gotPort != port {
			t.Errorf("port = %d, want %d", gotPort, port)
		}
		if v == nil || v.Identity == nil {
			t.Error("expected unlocked vault with identity")
		}
		select {
		case <-started:
		default:
			close(started)
		}
		<-ctx.Done()
		return nil
	}
	t.Cleanup(func() { runHTTPServerFunc = origHTTP })

	t.Cleanup(func() {
		_ = serveCmd.Flags().Set("bind", "127.0.0.1")
		_ = serveCmd.Flags().Set("stdio", "false")
		_ = serveCmd.Flags().Set("agent", "")
	})

	rootCmd.SetArgs([]string{"--vault", vaultDir, "serve", "--agent", "test-agent", "--port", fmt.Sprintf("%d", port)})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = rootCmd.Execute()
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP server did not start in time")
	}

	select {
	case sigCh := <-serveSignals:
		sigCh <- syscall.SIGTERM
	case <-time.After(2 * time.Second):
		t.Fatal("serve command did not install signal handler")
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("serve command did not shut down after test signal")
	}
}

func TestCmdServe_UninitializedVault(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)

	_ = serveCmd.Flags().Set("bind", "127.0.0.1")
	_ = serveCmd.Flags().Set("stdio", "false")

	rootCmd.SetArgs([]string{"--vault", vaultDir, "serve", "--bind", "127.0.0.1"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for uninitialized vault")
	}
	if !strings.Contains(execErr.Error(), "vault not initialized") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdMCPConfig_Stdio(t *testing.T) {
	vaultFlagReset(t)
	_ = mcpConfigCmd.Flags().Set("http", "false")
	t.Cleanup(func() { _ = mcpConfigCmd.Flags().Set("http", "false") })

	rootCmd.SetArgs([]string{"mcp-config", "myagent"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "openpass") {
		t.Errorf("mcp-config stdio output missing 'openpass': %q", output)
	}
	if !strings.Contains(output, "myagent") {
		t.Errorf("mcp-config stdio output missing agent name: %q", output)
	}
}

func TestCmdMCPConfig_StdioCustomVaultIncludesVaultArg(t *testing.T) {
	vaultDir := t.TempDir()
	vaultFlagReset(t)

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "--vault") || !strings.Contains(output, vaultDir) {
		t.Errorf("mcp-config stdio output for custom vault missing --vault arg: %q", output)
	}
}

func TestCmdMCPConfig_HTTP(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_VAULT") })

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "url") {
		t.Errorf("mcp-config http output missing 'url': %q", output)
	}
	if !strings.Contains(output, "Authorization") {
		t.Errorf("mcp-config http output missing 'Authorization': %q", output)
	}
}

func TestCmdMCPConfig_HermesHTTP(t *testing.T) {
	vaultDir, _ := initVault(t)
	vaultFlagReset(t)
	_ = mcpConfigCmd.Flags().Set("format", "generic")
	_ = mcpConfigCmd.Flags().Set("server-name", "openpass")
	t.Cleanup(func() {
		_ = mcpConfigCmd.Flags().Set("format", "generic")
		_ = mcpConfigCmd.Flags().Set("server-name", "openpass")
	})

	cfgContent := "mcp:\n  bind: 127.0.0.1\n  port: 8090\n"
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "hermes", "--http", "--format", "hermes"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	for _, want := range []string{
		"mcp_servers:",
		"openpass:",
		"url: http://127.0.0.1:8090/mcp",
		"Authorization: env:OPENPASS_MCP_TOKEN",
		`MCP-Protocol-Version: "2025-11-25"`,
		"X-OpenPass-Agent: hermes",
		"connect_timeout: 30",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("mcp-config hermes http output missing %q: %q", want, output)
		}
	}
}

func TestOutputHTTPConfig_VaultPathError(t *testing.T) {
	// Test outputHTTPConfig directly (bypassing rootCmd which has PersistentPreRun
	// that also calls vaultPath, causing a panic before our function is reached).
	origHome := os.Getenv("HOME")
	origVaultEnv := os.Getenv("OPENPASS_VAULT")
	origVault := vault
	origChanged := vaultFlag.Changed
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	vault = "~/.openpass"
	vaultFlag.Changed = false
	t.Cleanup(func() {
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("OPENPASS_VAULT", origVaultEnv)
		vault = origVault
		_ = vaultFlag.Value.Set(origVault)
		vaultFlag.Changed = origChanged
	})

	err := outputHTTPConfig("test-agent", "openpass", true)
	if err == nil {
		t.Error("expected error when HOME is unset for tilde expansion")
	}
}

func TestOutputHTTPConfig_CustomTokenFile(t *testing.T) {
	vaultDir, _ := initVault(t)
	vaultFlagReset(t)
	t.Cleanup(func() { _ = mcpConfigCmd.Flags().Set("include-token", "false") })

	customTokenPath := filepath.Join(t.TempDir(), "custom-token")
	customTokenValue := "my-custom-token-value-12345"
	if err := os.WriteFile(customTokenPath, []byte(customTokenValue+"\n"), 0o600); err != nil {
		t.Fatalf("write custom token: %v", err)
	}

	cfgContent := fmt.Sprintf("mcp:\n  bind: 127.0.0.1\n  port: 9999\n  httpTokenFile: %q\n", customTokenPath)
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "127.0.0.1:9999") {
		t.Errorf("mcp-config http output missing custom bind/port: %q", output)
	}
	if strings.Contains(output, customTokenValue) {
		t.Errorf("mcp-config http output leaked custom token without --include-token: %q", output)
	}
	if !strings.Contains(output, "env:OPENPASS_MCP_TOKEN") {
		t.Errorf("mcp-config http output missing redacted token reference: %q", output)
	}
	if !strings.Contains(output, "Authorization") {
		t.Errorf("mcp-config http output missing 'Authorization': %q", output)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http", "--include-token"})
	output = captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, customTokenValue) {
		t.Errorf("mcp-config http --include-token output missing custom token: %q", output)
	}
}

func TestOutputHTTPConfig_TokenLoadError(t *testing.T) {
	vaultDir, _ := initVault(t)
	vaultFlagReset(t)

	cfgContent := "mcp:\n  httpTokenFile: /nonexistent/path/mcp-token\n"
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for non-existent token path")
	}
	if !strings.Contains(execErr.Error(), "load token") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestOutputHTTPConfig_StaleRuntimePort(t *testing.T) {
	vaultDir, _ := initVault(t)
	vaultFlagReset(t)

	if err := saveRuntimePort(vaultDir, 1); err != nil {
		t.Fatalf("save runtime port: %v", err)
	}

	rootCmd.SetArgs([]string{"--vault", vaultDir, "mcp-config", "myagent", "--http"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Fatal("expected stale runtime port error")
	}
	if !strings.Contains(execErr.Error(), "stale runtime port") {
		t.Fatalf("unexpected error: %v", execErr)
	}
}

func TestCmdRecipientsRemove_WithYesFlag(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("add recipient: %v", err)
	}

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "remove", testRecipient1, "--yes"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "Recipient removed successfully") {
		t.Errorf("expected 'Recipient removed successfully', got: %q", output)
	}
}

func TestCmdRecipientsRemove_InvalidKey(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	_ = os.Setenv("OPENPASS_PASSPHRASE", passphrase)
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "remove", "not-a-valid-key", "--yes"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for invalid recipient key")
	}
	if !strings.Contains(execErr.Error(), "invalid recipient") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdRecipientsAdd_UnlockError(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	_ = os.Setenv("OPENPASS_PASSPHRASE", "wrong-passphrase")
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "add", testRecipient1})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for wrong passphrase")
	}
	if !strings.Contains(execErr.Error(), "open vault") {
		t.Errorf("unexpected error: %v", execErr)
	}
}

func TestCmdRecipientsRemove_UnlockError(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "correcthorsebatterystaple"
	vaultFlagReset(t)

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, config.Default()); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	_ = os.Setenv("OPENPASS_PASSPHRASE", "wrong-passphrase")
	t.Cleanup(func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") })

	rootCmd.SetArgs([]string{"--vault", vaultDir, "recipients", "remove", testRecipient1, "--yes"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var execErr error
	captureStderr(func() {
		execErr = rootCmd.Execute()
	})

	if execErr == nil {
		t.Error("expected error for wrong passphrase")
	}
	if !strings.Contains(execErr.Error(), "open vault") {
		t.Errorf("unexpected error: %v", execErr)
	}
}
