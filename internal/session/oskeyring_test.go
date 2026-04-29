//go:build darwin || linux || windows || netbsd || openbsd || ((freebsd || dragonfly) && cgo)

package session

import (
	"testing"
	"time"
)

func TestFallback_KeyringOperations(t *testing.T) {
	oldSet := keyringSet
	oldGet := keyringGet
	oldDelete := keyringDelete
	oldFallbackActive := fallbackActive
	oldFallback := fallback

	fallbackMu.Lock()
	fallbackActive = true
	fallback = &memoryKeyring{}
	fallbackMu.Unlock()

	keyringSet = setWithFallback
	keyringGet = getWithFallback
	keyringDelete = deleteWithFallback

	t.Cleanup(func() {
		keyringSet = oldSet
		keyringGet = oldGet
		keyringDelete = oldDelete
		fallbackMu.Lock()
		fallbackActive = oldFallbackActive
		fallback = oldFallback
		fallbackMu.Unlock()
	})

	vaultDir := "/tmp/vault-fallback"
	passphrase := "fallback-secret"

	if err := SavePassphrase(vaultDir, passphrase, time.Hour); err != nil {
		t.Fatalf("SavePassphrase() error = %v", err)
	}

	got, err := LoadPassphrase(vaultDir)
	if err != nil {
		t.Fatalf("LoadPassphrase() error = %v", err)
	}
	if got != passphrase {
		t.Errorf("LoadPassphrase() = %q, want %q", got, passphrase)
	}

	if err := ClearSession(vaultDir); err != nil {
		t.Fatalf("ClearSession() error = %v", err)
	}

	_, err = LoadPassphrase(vaultDir)
	if err == nil {
		t.Fatal("LoadPassphrase() after ClearSession error = nil, want not found")
	}
}

func TestFallback_Get_NotFound(t *testing.T) {
	oldGet := keyringGet
	oldFallbackActive := fallbackActive
	oldFallback := fallback

	fallbackMu.Lock()
	fallbackActive = true
	fallback = &memoryKeyring{}
	fallbackMu.Unlock()

	keyringGet = getWithFallback

	t.Cleanup(func() {
		keyringGet = oldGet
		fallbackMu.Lock()
		fallbackActive = oldFallbackActive
		fallback = oldFallback
		fallbackMu.Unlock()
	})

	_, err := LoadPassphrase("/tmp/vault-fallback-notfound")
	if err == nil {
		t.Fatal("LoadPassphrase() error = nil, want not found")
	}
}
