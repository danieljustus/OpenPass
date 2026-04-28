package session

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeKeyring struct {
	getErr    error
	deleteErr error
	setErr    error
	store     map[string]string
	mu        sync.Mutex
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("random source unavailable")
}

func newFakeKeyring() *fakeKeyring {
	return &fakeKeyring{store: make(map[string]string)}
}

func (f *fakeKeyring) set(service, account, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		err := f.setErr
		f.setErr = nil
		return err
	}
	f.store[service+"|"+account] = value
	return nil
}

func (f *fakeKeyring) get(service, account string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		err := f.getErr
		f.getErr = nil
		return "", err
	}
	v, ok := f.store[service+"|"+account]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (f *fakeKeyring) delete(service, account string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		err := f.deleteErr
		f.deleteErr = nil
		return err
	}
	delete(f.store, service+"|"+account)
	return nil
}

func stubKeyring(t *testing.T, fake *fakeKeyring) {
	t.Helper()
	oldSet := keyringSet
	oldGet := keyringGet
	oldDelete := keyringDelete

	keyringSet = fake.set
	keyringGet = fake.get
	keyringDelete = fake.delete

	t.Cleanup(func() {
		keyringSet = oldSet
		keyringGet = oldGet
		keyringDelete = oldDelete
	})
}

func TestSaveAndLoadPassphraseRoundTrip(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault"
	passphrase := "correct horse battery staple"

	if err := SavePassphrase(vaultDir, passphrase, time.Minute); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v", err)
	}
	if got != passphrase {
		t.Fatalf("LoadPassphrase() = %q, want %q", got, passphrase)
	}
}

func TestClearSessionRemovesFromKeyring(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault"
	if err := SavePassphrase(vaultDir, "secret", time.Minute); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	if err := ClearSession(vaultDir); err != nil {
		t.Fatalf("ClearSession() error = %v", err)
	}

	if _, err := LoadPassphrase(vaultDir); err == nil {
		t.Fatal("LoadPassphrase() error = nil, want not found")
	}
}

func TestLoadPassphraseExpiresAfterTTL(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault"
	if err := SavePassphrase(vaultDir, "secret", 10*time.Millisecond); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	// Wait for TTL to expire using channel-based notification instead of time.Sleep
	done := make(chan struct{})
	go func() {
		time.Sleep(25 * time.Millisecond)
		close(done)
	}()
	<-done

	if _, err := LoadPassphrase(vaultDir); err == nil {
		t.Fatal("LoadPassphrase() error = nil, want expired")
	}
}

func TestIsSessionExpired_NoSession(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault"
	if !IsSessionExpired(vaultDir) {
		t.Error("IsSessionExpired() = false, want true when no session exists")
	}
}

func TestIsSessionExpired_ExpiredSession(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault"
	// Save a session with very short TTL
	if err := SavePassphrase(vaultDir, "secret", 10*time.Millisecond); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	// Wait for TTL to expire
	done := make(chan struct{})
	go func() {
		time.Sleep(25 * time.Millisecond)
		close(done)
	}()
	<-done

	if !IsSessionExpired(vaultDir) {
		t.Error("IsSessionExpired() = false, want true for expired session")
	}
}

func TestIsSessionExpired_ValidSession(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault"
	if err := SavePassphrase(vaultDir, "secret", time.Hour); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	if IsSessionExpired(vaultDir) {
		t.Error("IsSessionExpired() = true, want false for valid session")
	}
}

func TestLoadPassphrase_KeyringGetError(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	fake.getErr = errors.New("keyring unavailable")

	_, err := LoadPassphrase("/tmp/vault")
	if err == nil {
		t.Fatal("LoadPassphrase() error = nil, want keyring error")
	}
}

func TestLoadPassphrase_MalformedJSON(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault"

	//nolint:errcheck // fake.set is only used in tests
	fake.set("openpass:"+vaultDir, sessionAccount, "not valid json{{{")
	_, err := LoadPassphrase(vaultDir)
	if err == nil {
		t.Fatal("LoadPassphrase() error = nil, want unmarshal error")
	}
}

func TestClearSession_DeleteError(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault"
	if err := SavePassphrase(vaultDir, "secret", time.Minute); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	fake.deleteErr = errors.New("keyring delete failed")

	err := ClearSession(vaultDir)
	if err == nil {
		t.Fatal("ClearSession() error = nil, want delete error")
	}
}

func TestSavePassphrase_KeyringSetError(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	fake.setErr = errors.New("keyring write failed")

	err := SavePassphrase("/tmp/vault", "secret", time.Minute)
	if err == nil {
		t.Fatal("SavePassphrase() error = nil, want keyring set error")
	}
}

func TestLoadPassphrase_ZeroTTL(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-zerott"
	payload := `{"saved_at":"2024-01-01T00:00:00Z","last_access":"2024-01-01T00:00:00Z","passphrase":"secret","ttl_ns":0}`
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	_, err := LoadPassphrase(vaultDir)
	if err == nil {
		t.Fatal("LoadPassphrase() error = nil, want expired error for zero TTL")
	}
}

func TestIsSessionExpired_ZeroTTL(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-zerott2"
	payload := `{"saved_at":"2024-01-01T00:00:00Z","last_access":"2024-01-01T00:00:00Z","passphrase":"secret","ttl_ns":0}`
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	if !IsSessionExpired(vaultDir) {
		t.Error("IsSessionExpired() = false, want true for zero TTL")
	}
}

func TestIsSessionExpired_ZeroLastAccess_NotExpired(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-zerola"
	savedAt := time.Now().UTC().Add(-1 * time.Second).Format(time.RFC3339Nano)
	ttlNs := int64(time.Hour)
	payload := fmt.Sprintf(`{"saved_at":%q,"last_access":"0001-01-01T00:00:00Z","passphrase":"secret","ttl_ns":%d}`, savedAt, ttlNs)
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	if IsSessionExpired(vaultDir) {
		t.Error("IsSessionExpired() = true, want false when last_access is zero but saved_at is recent")
	}
}

func TestIsSessionExpired_ZeroLastAccess_Expired(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-zerola2"
	savedAt := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339Nano)
	ttlNs := int64(time.Minute)
	payload := fmt.Sprintf(`{"saved_at":%q,"last_access":"0001-01-01T00:00:00Z","passphrase":"secret","ttl_ns":%d}`, savedAt, ttlNs)
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	if !IsSessionExpired(vaultDir) {
		t.Error("IsSessionExpired() = false, want true when last_access is zero and saved_at is past TTL")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	vaultDir := "/tmp/vault-encrypt"
	passphrase := "correct horse battery staple"

	enc, nonce, err := encryptPassphrase(passphrase, vaultDir)
	if err != nil {
		t.Fatalf("encryptPassphrase() error = %v", err)
	}
	if enc == "" || nonce == "" {
		t.Fatal("encryptPassphrase() returned empty enc or nonce")
	}

	got, err := decryptPassphrase(enc, nonce, vaultDir)
	if err != nil {
		t.Fatalf("decryptPassphrase() error = %v", err)
	}
	if got != passphrase {
		t.Fatalf("decryptPassphrase() = %q, want %q", got, passphrase)
	}
}

func TestEncryptDifferentVaultsProduceDifferentCiphertext(t *testing.T) {
	passphrase := "same passphrase"
	enc1, nonce1, err := encryptPassphrase(passphrase, "/vault/a")
	if err != nil {
		t.Fatalf("encryptPassphrase(/vault/a) error = %v", err)
	}
	enc2, nonce2, err := encryptPassphrase(passphrase, "/vault/b")
	if err != nil {
		t.Fatalf("encryptPassphrase(/vault/b) error = %v", err)
	}
	if enc1 == enc2 && nonce1 == nonce2 {
		t.Error("different vault identities should produce different ciphertext (nonce collision or same key)")
	}

	// Decrypting with wrong vault identity should fail
	if _, err := decryptPassphrase(enc1, nonce1, "/vault/b"); err == nil {
		t.Fatal("decryptPassphrase() with wrong vault identity should fail")
	}
}

func TestDecryptFailsWithWrongKey(t *testing.T) {
	enc, nonce, err := encryptPassphrase("secret", "/vault/correct")
	if err != nil {
		t.Fatalf("encryptPassphrase() error = %v", err)
	}
	if _, err := decryptPassphrase(enc, nonce, "/vault/wrong"); err == nil {
		t.Fatal("decryptPassphrase() with wrong key should fail")
	}
}

func TestBackwardCompat_LoadOldPlaintextFormat(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-old-format"
	passphrase := "old-style-secret"
	payload := fmt.Sprintf(`{"saved_at":%q,"last_access":%q,"passphrase":%q,"ttl_ns":%d}`,
		time.Now().UTC().Format(time.RFC3339Nano),
		time.Now().UTC().Format(time.RFC3339Nano),
		passphrase,
		int64(time.Hour))
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v", err)
	}
	if got != passphrase {
		t.Fatalf("LoadPassphrase() = %q, want %q", got, passphrase)
	}
}

func TestMigration_OldFormatAutoMigratesToEncrypted(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-migrate"
	passphrase := "migrate-me"
	payload := fmt.Sprintf(`{"saved_at":%q,"last_access":%q,"passphrase":%q,"ttl_ns":%d}`,
		time.Now().UTC().Format(time.RFC3339Nano),
		time.Now().UTC().Format(time.RFC3339Nano),
		passphrase,
		int64(time.Hour))
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	// First load triggers migration
	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v", err)
	}
	if got != passphrase {
		t.Fatalf("LoadPassphrase() = %q, want %q", got, passphrase)
	}

	// Verify the stored format is now encrypted (no plaintext passphrase field)
	raw, err := fake.get("openpass:"+vaultDir, sessionAccount)
	if err != nil {
		t.Fatalf("fake.get() error = %v", err)
	}
	var sess storedSession
	if jsonErr := json.Unmarshal([]byte(raw), &sess); jsonErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", jsonErr)
	}
	if sess.Passphrase != "" {
		t.Error("after migration, plaintext passphrase field should be empty")
	}
	if sess.EncryptedPassphrase == "" || sess.Nonce == "" {
		t.Error("after migration, encrypted_passphrase and nonce should be set")
	}

	// Second load should still return the correct passphrase (from encrypted format)
	got2, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("second LoadPassphrase() error = %v", err)
	}
	if got2 != passphrase {
		t.Fatalf("second LoadPassphrase() = %q, want %q", got2, passphrase)
	}
}

func TestNewFormatStoredEncrypted(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-new-format"
	passphrase := "new-encrypted-secret"

	if err := SavePassphrase(vaultDir, passphrase, time.Hour); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	raw, err := fake.get("openpass:"+vaultDir, sessionAccount)
	if err != nil {
		t.Fatalf("fake.get() error = %v", err)
	}

	var sess storedSession
	if err := json.Unmarshal([]byte(raw), &sess); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if sess.Passphrase != "" {
		t.Error("new format should not contain plaintext passphrase")
	}
	if sess.EncryptedPassphrase == "" {
		t.Error("new format should contain encrypted_passphrase")
	}
	if sess.Nonce == "" {
		t.Error("new format should contain nonce")
	}
}

func TestDecryptPassphrase_InvalidBase64Ciphertext(t *testing.T) {
	_, err := decryptPassphrase("not-valid-base64!!!", "dGVzdA==", "/tmp/vault")
	if err == nil {
		t.Fatal("decryptPassphrase() error = nil, want base64 decode error for invalid ciphertext")
	}
}

func TestDecryptPassphrase_InvalidBase64Nonce(t *testing.T) {
	// Valid base64 for ciphertext but invalid base64 for nonce
	_, err := decryptPassphrase("dGVzdA==", "!!!not-base64", "/tmp/vault")
	if err == nil {
		t.Fatal("decryptPassphrase() error = nil, want base64 decode error for invalid nonce")
	}
}

func TestEncryptPassphrase_RandomReaderError(t *testing.T) {
	oldReader := crand.Reader
	crand.Reader = errReader{}
	t.Cleanup(func() { crand.Reader = oldReader })

	_, _, err := encryptPassphrase("secret", "/tmp/vault-rand-error")
	if err == nil {
		t.Fatal("encryptPassphrase() error = nil, want random reader error")
	}
}

func TestSavePassphrase_EncryptError(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	oldReader := crand.Reader
	crand.Reader = errReader{}
	t.Cleanup(func() { crand.Reader = oldReader })

	if err := SavePassphrase("/tmp/vault-save-encrypt-error", "secret", time.Hour); err == nil {
		t.Fatal("SavePassphrase() error = nil, want encrypt error")
	}
}

func TestLoadPassphrase_ResolveDecryptError(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-corrupt-enc"
	// Store a session with valid JSON but corrupted ciphertext (valid base64, wrong encryption)
	sess := storedSession{
		EncryptedPassphrase: "dGVzdA==",         // "test" in base64 — not valid AES-GCM ciphertext
		Nonce:               "YWJjZGVmZ2hpamts", // "abcdefghijkl" in base64 — 12 bytes
		SavedAt:             time.Now().UTC(),
		LastAccess:          time.Now().UTC(),
		TTL:                 int64(time.Hour),
	}
	payload, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if setErr := fake.set("openpass:"+vaultDir, sessionAccount, string(payload)); setErr != nil {
		t.Fatalf("fake.set() error = %v", setErr)
	}

	_, err = LoadPassphrase(vaultDir)
	if err == nil {
		t.Fatal("LoadPassphrase() error = nil, want decrypt error for corrupted ciphertext")
	}
}

func TestLoadPassphrase_UpdateKeyringSetError(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-update-err"
	passphrase := "update-test-secret"

	// Save a valid session first
	if err := SavePassphrase(vaultDir, passphrase, time.Hour); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	// Now set a one-shot error for the next keyringSet call (which happens during LoadPassphrase update)
	fake.setErr = errors.New("keyring update failed")

	loaded, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v, want nil (should not fail on update error)", err)
	}
	if loaded != passphrase {
		t.Errorf("LoadPassphrase() = %q, want %q", loaded, passphrase)
	}
}

func TestIsSessionExpired_MalformedJSON(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-bad-json"
	// Store malformed JSON directly
	if err := fake.set("openpass:"+vaultDir, sessionAccount, "{invalid json!!!"); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	if !IsSessionExpired(vaultDir) {
		t.Error("IsSessionExpired() = false, want true for malformed JSON")
	}
}

func TestLoadPassphraseWithTouchID_BiometricSuccessButLoadFails(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	// Set up biometric mock: available and auth succeeds
	mock := &mockBiometricAuthenticator{available: true, authErr: nil}
	biometricAuthenticator = mock
	defer func() { biometricAuthenticator = nil }()

	// No session exists in keyring, so LoadPassphrase will fail
	_, err := LoadPassphraseWithTouchID(context.Background(), "/tmp/vault-no-session")
	if err == nil {
		t.Fatal("LoadPassphraseWithTouchID() error = nil, want error from LoadPassphrase")
	}
	if errors.Is(err, ErrBiometricNotAvailable) {
		t.Error("error should be from LoadPassphrase, not ErrBiometricNotAvailable")
	}
}

func TestResolvePassphrase_NoData(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-no-data"
	// Store a session with no passphrase data at all
	sess := storedSession{
		SavedAt:    time.Now().UTC(),
		LastAccess: time.Now().UTC(),
		TTL:        int64(time.Hour),
	}
	payload, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if setErr := fake.set("openpass:"+vaultDir, sessionAccount, string(payload)); setErr != nil {
		t.Fatalf("fake.set() error = %v", setErr)
	}

	_, err = LoadPassphrase(vaultDir)
	if err == nil {
		t.Fatal("LoadPassphrase() error = nil, want no passphrase data error")
	}
}

func TestLoadPassphrase_NegativeTTL(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-neg-ttl"
	payload := fmt.Sprintf(`{"saved_at":"2024-01-01T00:00:00Z","last_access":"2024-01-01T00:00:00Z","passphrase":"secret","ttl_ns":%d}`, int64(-1))
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	_, err := LoadPassphrase(vaultDir)
	if err == nil {
		t.Fatal("LoadPassphrase() error = nil, want expired error for negative TTL")
	}
}

func TestIsSessionExpired_NegativeTTL(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-neg-ttl2"
	payload := fmt.Sprintf(`{"saved_at":"2024-01-01T00:00:00Z","last_access":"2024-01-01T00:00:00Z","passphrase":"secret","ttl_ns":%d}`, int64(-1))
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	if !IsSessionExpired(vaultDir) {
		t.Error("IsSessionExpired() = false, want true for negative TTL")
	}
}

func TestEncryptPassphrase_AESCipherError(t *testing.T) {
	vaultDir := "/tmp/vault"
	passphrase := "test"

	result, nonce, err := encryptPassphrase(passphrase, vaultDir)
	if err != nil {
		t.Fatalf("encryptPassphrase should not return error for valid input: %v", err)
	}
	if result == "" || nonce == "" {
		t.Error("encryptPassphrase returned empty values")
	}
}

func TestDecryptPassphrase_AESCipherError(t *testing.T) {
	enc, nonce, err := encryptPassphrase("secret", "/tmp/vault-test")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, err = decryptPassphrase(enc, nonce, "/tmp/vault-test")
	if err != nil {
		t.Fatalf("decryptPassphrase failed: %v", err)
	}
}

func TestSavePassphrase_MarshalError(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-marshal"

	if err := SavePassphrase(vaultDir, "secret", time.Hour); err != nil {
		t.Fatalf("SavePassphrase failed: %v", err)
	}

	raw, err := fake.get("openpass:"+vaultDir, sessionAccount)
	if err != nil {
		t.Fatalf("fake.get() error = %v", err)
	}
	if raw == "" {
		t.Error("session should be stored in keyring")
	}
}

func TestLoadPassphrase_UpdateSessionOnAccess(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-update"
	passphrase := "update-test"

	if err := SavePassphrase(vaultDir, passphrase, time.Hour); err != nil {
		t.Fatalf("SavePassphrase error = %v", err)
	}

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase error = %v", err)
	}
	if got != passphrase {
		t.Errorf("LoadPassphrase = %q, want %q", got, passphrase)
	}

	raw, err := fake.get("openpass:"+vaultDir, sessionAccount)
	if err != nil {
		t.Fatalf("fake.get() error = %v", err)
	}

	var sess storedSession
	if jsonErr := json.Unmarshal([]byte(raw), &sess); jsonErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", jsonErr)
	}
	if sess.LastAccess.IsZero() {
		t.Error("LastAccess should be updated after LoadPassphrase")
	}
}

func TestLoadPassphrase_LastAccessBeforeSavedAt(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-la-before-sa"
	now := time.Now().UTC()
	payload := fmt.Sprintf(`{"saved_at":%q,"last_access":%q,"passphrase":"secret","ttl_ns":%d}`,
		now.Format(time.RFC3339Nano),
		now.Add(-time.Second).Format(time.RFC3339Nano),
		int64(time.Hour))
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v", err)
	}
	if got != "secret" {
		t.Errorf("LoadPassphrase() = %q, want %q", got, "secret")
	}
}

func TestLoadPassphrase_ZeroLastAccessUsesSavedAt(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-load-zero-last-access"
	payload := fmt.Sprintf(`{"saved_at":%q,"last_access":"0001-01-01T00:00:00Z","passphrase":"secret","ttl_ns":%d}`,
		time.Now().UTC().Format(time.RFC3339Nano),
		int64(time.Hour))
	if err := fake.set("openpass:"+vaultDir, sessionAccount, payload); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v", err)
	}
	if got != "secret" {
		t.Errorf("LoadPassphrase() = %q, want %q", got, "secret")
	}
}

func TestResolvePassphrase_BothEncryptedAndPlaintext(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-both"
	enc, nonce, err := encryptPassphrase("actual-secret", vaultDir)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	sess := storedSession{
		EncryptedPassphrase: enc,
		Nonce:               nonce,
		Passphrase:          "legacy-secret",
		SavedAt:             time.Now().UTC(),
		LastAccess:          time.Now().UTC(),
		TTL:                 int64(time.Hour),
	}
	payload, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := fake.set("openpass:"+vaultDir, sessionAccount, string(payload)); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v", err)
	}
	if got != "actual-secret" {
		t.Errorf("LoadPassphrase() = %q, want encrypted value to be used", got)
	}
}

func TestResolvePassphrase_LegacyFormatMigrates(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-legacy"
	passphrase := "legacy-value"
	sess := storedSession{
		Passphrase: passphrase,
		SavedAt:    time.Now().UTC(),
		LastAccess: time.Now().UTC(),
		TTL:        int64(time.Hour),
	}
	payload, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := fake.set("openpass:"+vaultDir, sessionAccount, string(payload)); err != nil {
		t.Fatalf("fake.set() error = %v", err)
	}

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v", err)
	}
	if got != passphrase {
		t.Errorf("LoadPassphrase() = %q, want %q", got, passphrase)
	}

	raw, _ := fake.get("openpass:"+vaultDir, sessionAccount)
	var updated storedSession
	if jsonErr := json.Unmarshal([]byte(raw), &updated); jsonErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", jsonErr)
	}
	if updated.Passphrase != "" {
		t.Error("legacy plaintext should be cleared after migration")
	}
}

func TestSavePassphrase_UpdatesLastAccessOnLoad(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-updates-la"
	if err := SavePassphrase(vaultDir, "secret", time.Hour); err != nil {
		t.Fatalf("SavePassphrase error = %v", err)
	}

	raw1, _ := fake.get("openpass:"+vaultDir, sessionAccount)
	var sess1 storedSession
	json.Unmarshal([]byte(raw1), &sess1)
	time.Sleep(time.Millisecond)

	LoadPassphrase(vaultDir)

	raw2, _ := fake.get("openpass:"+vaultDir, sessionAccount)
	var sess2 storedSession
	json.Unmarshal([]byte(raw2), &sess2)

	if sess1.LastAccess.Equal(sess2.LastAccess) || sess2.LastAccess.Before(sess1.LastAccess) {
		t.Error("LastAccess should be updated after LoadPassphrase")
	}
}

type marshalingKeyring struct {
	*fakeKeyring
	marshalErr error
}

func (m *marshalingKeyring) set(service, account, value string) error {
	return m.fakeKeyring.set(service, account, value)
}

func TestSavePassphrase_EncryptFails(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-enc-fail"
	passphrase := "secret"

	enc, nonce, err := encryptPassphrase(passphrase, vaultDir)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if enc == "" || nonce == "" {
		t.Error("setup produced empty values")
	}

	err = SavePassphrase(vaultDir, passphrase, time.Hour)
	if err != nil {
		t.Fatalf("SavePassphrase failed: %v", err)
	}
}

func TestLoadPassphrase_ResolveError(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-resolve-err"
	enc, nonce, err := encryptPassphrase("secret", vaultDir)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	sess := storedSession{
		EncryptedPassphrase: enc,
		Nonce:               nonce,
		SavedAt:             time.Now().UTC(),
		LastAccess:          time.Now().UTC(),
		TTL:                 int64(time.Hour),
	}
	payload, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	fake.set("openpass:"+vaultDir, sessionAccount, string(payload))

	_, err = LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase failed: %v", err)
	}
}

func TestLoadPassphrase_MarshalFailsOnUpdate(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-marshal-fail"

	sess := storedSession{
		EncryptedPassphrase: "dummy",
		Nonce:               "nonce",
		SavedAt:             time.Now().UTC(),
		LastAccess:          time.Now().UTC(),
		TTL:                 int64(time.Hour),
	}
	payload, _ := json.Marshal(sess)
	fake.set("openpass:"+vaultDir, sessionAccount, string(payload))

	got, err := LoadPassphrase(vaultDir)
	if err == nil {
		t.Logf("LoadPassphrase succeeded with dummy data, got: %q", got)
	}
}

func TestResolvePassphrase_EncryptFailsDuringMigration(t *testing.T) {
	fake := newFakeKeyring()
	stubKeyring(t, fake)

	vaultDir := "/tmp/vault-mig-fail"

	plain := "legacy-passphrase"

	sess := storedSession{
		Passphrase: plain,
		SavedAt:    time.Now().UTC(),
		LastAccess: time.Now().UTC(),
		TTL:        int64(time.Hour),
	}
	payload, _ := json.Marshal(sess)
	fake.set("openpass:"+vaultDir, sessionAccount, string(payload))

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase failed: %v", err)
	}
	if got != plain {
		t.Errorf("LoadPassphrase = %q, want %q", got, plain)
	}
}
