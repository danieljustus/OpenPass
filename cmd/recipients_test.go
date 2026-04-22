package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/testutil"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

const (
	testRecipient1 = "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"
	testRecipient2 = "age1savzx9za5xg4fvwkeq788v50esvs3ccn9sscdxevw2fev9xdyeps8z9z65"
)

func TestRecipientsListCmd_VaultNotInitialized(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	vault = vaultDir

	cmd := recipientsListCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	origArgs := os.Args
	os.Args = []string{"openpass", "recipients", "list"}
	defer func() { os.Args = origArgs }()

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for vault not initialized")
	}
	if !strings.Contains(err.Error(), "vault not initialized") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRecipientsAddCmd_VaultNotInitialized(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	vault = vaultDir

	cmd := recipientsAddCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	origArgs := os.Args
	os.Args = []string{"openpass", "recipients", "add", testRecipient1}
	defer func() { os.Args = origArgs }()

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for vault not initialized")
	}
}

func TestRecipientsRemoveCmd_VaultNotInitialized(t *testing.T) {
	vaultDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", vaultDir)
	defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

	vault = vaultDir

	cmd := recipientsRemoveCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	origArgs := os.Args
	os.Args = []string{"openpass", "recipients", "remove", testRecipient1}
	defer func() { os.Args = origArgs }()

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for vault not initialized")
	}
}

func TestListRecipients_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	recipients, err := rm.ListRecipients()
	if err != nil {
		t.Fatalf("failed to list recipients: %v", err)
	}
	if len(recipients) != 0 {
		t.Errorf("expected 0 recipients, got %d", len(recipients))
	}
}

func TestRecipientsManagerInvalidKey_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	err := rm.AddRecipient("invalid-key")
	if err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestRecipientsManagerRemoveNotFound_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	err := rm.RemoveRecipient("age1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err == nil {
		t.Error("expected error for removing non-existent recipient")
	}
}

func TestRecipientsList_WithRecipients_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("failed to add recipient: %v", err)
	}
	if err := rm.AddRecipient(testRecipient2); err != nil {
		t.Fatalf("failed to add recipient: %v", err)
	}

	recipients, err := rm.ListRecipients()
	if err != nil {
		t.Fatalf("failed to list recipients: %v", err)
	}
	if len(recipients) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(recipients))
	}

	for _, r := range recipients {
		if !r.Valid {
			t.Errorf("expected recipient to be valid: %s", r.RawString)
		}
	}
}

func TestRecipientsAdd_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	err := rm.AddRecipient(testRecipient1)
	if err != nil {
		t.Fatalf("failed to add recipient: %v", err)
	}

	recipients, err := rm.ListRecipients()
	if err != nil {
		t.Fatalf("failed to list recipients: %v", err)
	}
	if len(recipients) != 1 {
		t.Errorf("expected 1 recipient, got %d", len(recipients))
	}
}

func TestRecipientsAdd_Duplicate_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("failed to add recipient: %v", err)
	}

	err := rm.AddRecipient(testRecipient1)
	if err == nil {
		t.Error("expected error for duplicate recipient")
	}
	if !errors.Is(err, vaultpkg.ErrRecipientAlreadyExists) {
		t.Errorf("expected ErrRecipientAlreadyExists, got: %v", err)
	}
}

func TestRecipientsRemove_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("failed to add recipient: %v", err)
	}

	err := rm.RemoveRecipient(testRecipient1)
	if err != nil {
		t.Fatalf("failed to remove recipient: %v", err)
	}

	recipients, err := rm.ListRecipients()
	if err != nil {
		t.Fatalf("failed to list recipients: %v", err)
	}
	if len(recipients) != 0 {
		t.Errorf("expected 0 recipients, got %d", len(recipients))
	}
}

func TestMultiUserEncryption_Integration(t *testing.T) {
	vaultDir := t.TempDir()

	identity1 := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity1, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("failed to add recipient 1: %v", err)
	}
	if err := rm.AddRecipient(testRecipient2); err != nil {
		t.Fatalf("failed to add recipient 2: %v", err)
	}

	entry := &vaultpkg.Entry{
		Data: map[string]any{
			"password": "shared-secret",
			"username": "shared-user",
		},
	}
	if err := vaultpkg.WriteEntryWithRecipients(vaultDir, "shared-entry", entry, identity1); err != nil {
		t.Fatalf("failed to write entry: %v", err)
	}

	entryPath := filepath.Join(vaultDir, "entries", "shared-entry.age")
	if _, err := os.Stat(entryPath); os.IsNotExist(err) {
		t.Fatal("entry file was not created")
	}

	readEntry, err := vaultpkg.ReadEntry(vaultDir, "shared-entry", identity1)
	if err != nil {
		t.Fatalf("failed to read entry with identity1: %v", err)
	}
	if readEntry.Data["password"] != "shared-secret" {
		t.Errorf("expected password 'shared-secret', got %v", readEntry.Data["password"])
	}

	v := &vaultpkg.Vault{Dir: vaultDir, Identity: identity1}
	recipients, err := v.GetAllRecipientsForEncryption()
	if err != nil {
		t.Fatalf("failed to get recipients: %v", err)
	}

	if len(recipients) != 3 {
		t.Errorf("expected 3 recipients, got %d", len(recipients))
	}
}

func TestRecipientsList_ValidAndInvalid_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	recipientsPath := filepath.Join(vaultDir, "recipients.txt")
	content := testRecipient1 + "\ninvalid-key\n" + testRecipient2 + "\n"
	if err := os.WriteFile(recipientsPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write recipients file: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	recipients, err := rm.ListRecipients()
	if err != nil {
		t.Fatalf("failed to list recipients: %v", err)
	}
	if len(recipients) != 3 {
		t.Errorf("expected 3 recipients, got %d", len(recipients))
	}

	validCount := 0
	invalidCount := 0
	for _, r := range recipients {
		if r.Valid {
			validCount++
		} else {
			invalidCount++
		}
	}
	if validCount != 2 {
		t.Errorf("expected 2 valid recipients, got %d", validCount)
	}
	if invalidCount != 1 {
		t.Errorf("expected 1 invalid recipient, got %d", invalidCount)
	}
}

func TestRecipientsAdd_Multiple_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("failed to add recipient 1: %v", err)
	}
	if err := rm.AddRecipient(testRecipient2); err != nil {
		t.Fatalf("failed to add recipient 2: %v", err)
	}

	recipients, err := rm.ListRecipients()
	if err != nil {
		t.Fatalf("failed to list recipients: %v", err)
	}
	if len(recipients) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(recipients))
	}
}

func TestRecipientsRemove_CorrectOne_Integration(t *testing.T) {
	vaultDir := t.TempDir()
	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	cfg.VaultDir = vaultDir
	if err := vaultpkg.Init(vaultDir, identity, cfg); err != nil {
		t.Fatalf("failed to init vault: %v", err)
	}

	rm := vaultpkg.NewRecipientsManager(vaultDir)
	if err := rm.AddRecipient(testRecipient1); err != nil {
		t.Fatalf("failed to add recipient 1: %v", err)
	}
	if err := rm.AddRecipient(testRecipient2); err != nil {
		t.Fatalf("failed to add recipient 2: %v", err)
	}

	err := rm.RemoveRecipient(testRecipient1)
	if err != nil {
		t.Fatalf("failed to remove recipient: %v", err)
	}

	recipients, err := rm.ListRecipients()
	if err != nil {
		t.Fatalf("failed to list recipients: %v", err)
	}
	if len(recipients) != 1 {
		t.Errorf("expected 1 recipient after removal, got %d", len(recipients))
	}

	if len(recipients) != 1 || !strings.Contains(recipients[0].RawString, testRecipient2) {
		t.Errorf("expected only recipient 2 to remain, got: %v", recipients)
	}
}
