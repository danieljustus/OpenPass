package vault

import (
	"os"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/testutil"
)

// TestOpenWithPathTraversal covers validateVaultDir error path + Open error return.
func TestOpenWithPathTraversal(t *testing.T) {
	identity := testutil.TempIdentity(t)
	_, err := Open("../evil-vault", identity)
	if err == nil {
		t.Fatal("Open() error = nil, want ErrVaultDirEscapes for path traversal")
	}
}

// TestOpenWithMissingConfig covers the config load error path in Open.
func TestOpenWithMissingConfig(t *testing.T) {
	dir := t.TempDir() // exists but has no config.yaml
	identity := testutil.TempIdentity(t)
	_, err := Open(dir, identity)
	if err == nil {
		t.Fatal("Open() error = nil, want error when config.yaml is missing")
	}
}

// TestOpenWithPassphrasePathTraversal covers validateVaultDir error in OpenWithPassphrase.
func TestOpenWithPassphrasePathTraversal(t *testing.T) {
	_, err := OpenWithPassphrase("../evil-vault", "passphrase")
	if err == nil {
		t.Fatal("OpenWithPassphrase() error = nil, want ErrVaultDirEscapes")
	}
}

// TestVaultInitMkdirAllError covers the os.MkdirAll failure branch in vault.Init.
func TestVaultInitMkdirAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod 0 has no effect")
	}
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	defer os.Chmod(parent, 0o700) //nolint:errcheck

	identity := testutil.TempIdentity(t)
	cfg := config.Default()
	err := Init(parent+"/new-vault", identity, cfg)
	if err == nil {
		t.Fatal("Init() error = nil, want error when parent dir is not writable")
	}
}

// TestUnmarshalJSONInvalid covers the json.Unmarshal error path in Entry.UnmarshalJSON.
func TestUnmarshalJSONInvalid(t *testing.T) {
	var e Entry
	if err := e.UnmarshalJSON([]byte("this is not json {")); err == nil {
		t.Fatal("UnmarshalJSON() error = nil, want error for invalid JSON")
	}
}

// TestDeleteEntryNotFound covers os.Remove error when the file doesn't exist.
func TestDeleteEntryNotFound(t *testing.T) {
	vaultDir := t.TempDir()
	err := DeleteEntry(vaultDir, "nonexistent/entry")
	if err == nil {
		t.Fatal("DeleteEntry() error = nil, want error for non-existent entry")
	}
}

// TestMergeEntryNotFound covers the ReadEntry error path in MergeEntry.
func TestMergeEntryNotFound(t *testing.T) {
	vaultDir := t.TempDir()
	id := testutil.TempIdentity(t)
	_, err := MergeEntry(vaultDir, "nonexistent/entry", map[string]any{"key": "val"}, id)
	if err == nil {
		t.Fatal("MergeEntry() error = nil, want error for non-existent entry")
	}
}

// TestInitWithPassphraseMkdirAllError covers the os.MkdirAll failure in InitWithPassphrase.
func TestInitWithPassphraseMkdirAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod 0 has no effect")
	}
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	defer os.Chmod(parent, 0o700) //nolint:errcheck

	_, err := InitWithPassphrase(parent+"/new-vault", "passphrase", config.Default())
	if err == nil {
		t.Fatal("InitWithPassphrase() error = nil, want error when parent dir is not writable")
	}
}

// TestCollectFieldMatchesPrefixlessScalar covers the prefix=="" early-return branch.
func TestCollectFieldMatchesPrefixlessScalar(t *testing.T) {
	matches := make(map[string]struct{})
	// A scalar value with empty prefix should be skipped (not added to matches)
	CollectFieldMatches(matches, "", "just-a-scalar", "just")
	if len(matches) != 0 {
		t.Errorf("expected no matches for prefix-less scalar, got %v", matches)
	}
}

// TestGetRecipientNilVaultPointer covers the v==nil branch of GetRecipient.
func TestGetRecipientNilVaultPointer(t *testing.T) {
	var v *Vault
	_, err := v.GetRecipient()
	if err == nil {
		t.Fatal("GetRecipient() on nil vault should return error")
	}
}

// TestRememberSearchIdentityNilSkips covers the nil-identity early return.
func TestRememberSearchIdentityNilSkips(t *testing.T) {
	// Should not panic and should not overwrite existing identity
	id := testutil.TempIdentity(t)
	rememberSearchIdentity(id)
	rememberSearchIdentity(nil)
	got := currentSearchIdentity()
	if got == nil {
		t.Error("rememberSearchIdentity(nil) should not overwrite existing identity")
	}
}

// TestFindListError covers the List error return path in Find.
func TestFindListError(t *testing.T) {
	// vaultDir does not exist → List will fail → Find returns error
	_, err := Find("/nonexistent/vault/dir/that/does/not/exist", "query")
	if err == nil {
		t.Fatal("Find() error = nil, want error when vault dir does not exist")
	}
}

// TestMergeEntryWithRecipientsNotFound covers the ReadEntry error path in MergeEntryWithRecipients.
func TestMergeEntryWithRecipientsNotFoundCoverage(t *testing.T) {
	vaultDir := t.TempDir()
	id := testutil.TempIdentity(t)
	_, err := MergeEntryWithRecipients(vaultDir, "nonexistent/entry", map[string]any{"k": "v"}, id)
	if err == nil {
		t.Fatal("MergeEntryWithRecipients() error = nil, want error for non-existent entry")
	}
}
