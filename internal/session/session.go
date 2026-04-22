// Package session provides secure passphrase caching via OS keyring.
//
// Security Model:
//
// This package uses zalando/go-keyring to store vault passphrases in the
// OS keyring (macOS Keychain, Linux GNOME Keyring via D-Bus Secret Service,
// or Windows Credential Manager). The security properties are:
//
//  1. Encryption at Rest: All secrets are encrypted at rest by the OS keyring
//     using AES-256 (macOS Keychain) or equivalent mechanisms.
//
// 2. Transport Security:
//
//   - macOS: Secret passed via stdin to /usr/bin/security CLI (not visible in ps)
//
//   - Linux: D-Bus Secret Service API transmits secret as bytes. D-Bus is
//     local IPC; same-user processes can typically access session bus.
//
//   - Windows: Credential Manager API
//
//     3. Access Control: OS keyring requires user authentication to unlock.
//     The keyring typically prompts for password on first access per session.
//
// Threat Model Considerations:
//
//   - Local user access: OS keyring provides appropriate protection against
//     other local users (file permissions, user-specific keyring).
//   - Memory exposure: Passphrase exists in process memory during keyring
//     operations - unavoidable with any keyring integration.
//   - D-Bus interception (Linux): D-Bus is not encrypted by default for
//     local IPC. However, accessing D-Bus secrets requires the same user or
//     specific system configuration. If an attacker can sniff D-Bus messages,
//     they typically already have equivalent access to the user's session.
//
// Application-Level Encryption:
//
// In addition to OS keyring encryption, passphrases are encrypted with
// AES-256-GCM before keyring storage. The encryption key is derived from
// the vault directory path using PBKDF2-SHA256 (600,000 iterations).
// This provides defense-in-depth: even if the keyring blob is extracted,
// the passphrase remains encrypted without knowledge of the vault path.
//
// Backward Compatibility:
//
// Sessions stored in the legacy plaintext format (with a "passphrase" JSON
// field) are still readable. On load, old-format sessions are automatically
// re-encrypted and saved in the new format.
//
// See: https://github.com/zalando/go-keyring for library details.
package session

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"crypto/sha256"

	"github.com/zalando/go-keyring"
)

const (
	sessionAccount = "session"
	pbkdf2Iter     = 600_000
	pbkdf2KeyLen   = 32
	aesGCMNonceLen = 12
)

var (
	keyringSet    = keyring.Set
	keyringGet    = keyring.Get
	keyringDelete = keyring.Delete
)

type storedSession struct {
	SavedAt             time.Time `json:"saved_at"`
	LastAccess          time.Time `json:"last_access"`
	Passphrase          string    `json:"passphrase,omitempty"`
	EncryptedPassphrase string    `json:"encrypted_passphrase,omitempty"`
	Nonce               string    `json:"nonce,omitempty"`
	TTL                 int64     `json:"ttl_ns"`
}

func deriveKey(vaultIdentity string) []byte {
	return pbkdf2.Key([]byte(vaultIdentity), []byte("openpass-session"), pbkdf2Iter, pbkdf2KeyLen, sha256.New)
}

func encryptPassphrase(plaintext string, vaultIdentity string) (string, string, error) {
	key := deriveKey(vaultIdentity)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, aesGCMNonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), base64.StdEncoding.EncodeToString(nonce), nil
}

func decryptPassphrase(encB64, nonceB64, vaultIdentity string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encB64)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	key := deriveKey(vaultIdentity)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

func serviceName(vaultDir string) string {
	return "openpass:" + vaultDir
}

func SavePassphrase(vaultDir string, passphrase string, ttl time.Duration) error {
	now := time.Now().UTC()
	enc, nonce, encErr := encryptPassphrase(passphrase, vaultDir)
	if encErr != nil {
		return fmt.Errorf("encrypt passphrase: %w", encErr)
	}
	payload, err := json.Marshal(storedSession{
		EncryptedPassphrase: enc,
		Nonce:               nonce,
		SavedAt:             now,
		LastAccess:          now,
		TTL:                 int64(ttl),
	})
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := keyringSet(serviceName(vaultDir), sessionAccount, string(payload)); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

func LoadPassphrase(vaultDir string) (string, error) {
	raw, err := keyringGet(serviceName(vaultDir), sessionAccount)
	if err != nil {
		return "", fmt.Errorf("load session: %w", err)
	}

	var sess storedSession
	if unmarshalErr := json.Unmarshal([]byte(raw), &sess); unmarshalErr != nil {
		return "", fmt.Errorf("decode session: %w", unmarshalErr)
	}

	if sess.TTL <= 0 {
		return "", errors.New("session expired")
	}

	lastActivity := sess.LastAccess
	if lastActivity.IsZero() {
		lastActivity = sess.SavedAt
	}
	if time.Since(lastActivity) > time.Duration(sess.TTL) {
		_ = ClearSession(vaultDir)
		return "", errors.New("session expired")
	}

	passphrase, resolveErr := resolvePassphrase(&sess, vaultDir)
	if resolveErr != nil {
		return "", resolveErr
	}

	sess.LastAccess = time.Now().UTC()
	payload, err := json.Marshal(sess)
	if err != nil {
		return passphrase, nil
	}
	if updateErr := keyringSet(serviceName(vaultDir), sessionAccount, string(payload)); updateErr != nil {
		return "", fmt.Errorf("update session last access: %w", updateErr)
	}

	return passphrase, nil
}

func resolvePassphrase(sess *storedSession, vaultDir string) (string, error) {
	if sess.EncryptedPassphrase != "" && sess.Nonce != "" {
		plain, err := decryptPassphrase(sess.EncryptedPassphrase, sess.Nonce, vaultDir)
		if err != nil {
			return "", fmt.Errorf("decrypt session: %w", err)
		}
		return plain, nil
	}

	if sess.Passphrase != "" {
		plain := sess.Passphrase
		enc, nonce, encErr := encryptPassphrase(plain, vaultDir)
		if encErr == nil {
			sess.EncryptedPassphrase = enc
			sess.Nonce = nonce
			sess.Passphrase = ""
		}
		return plain, nil
	}

	return "", errors.New("session expired")
}

func ClearSession(vaultDir string) error {
	if err := keyringDelete(serviceName(vaultDir), sessionAccount); err != nil {
		return fmt.Errorf("clear session: %w", err)
	}
	return nil
}

func IsSessionExpired(vaultDir string) bool {
	raw, err := keyringGet(serviceName(vaultDir), sessionAccount)
	if err != nil {
		return true
	}

	var sess storedSession
	if err := json.Unmarshal([]byte(raw), &sess); err != nil {
		return true
	}

	if sess.TTL <= 0 {
		return true
	}

	lastActivity := sess.LastAccess
	if lastActivity.IsZero() {
		lastActivity = sess.SavedAt
	}
	return time.Since(lastActivity) > time.Duration(sess.TTL)
}

func LoadPassphraseWithTouchID(ctx context.Context, vaultDir string) (string, error) {
	biometric := DefaultBiometricAuthenticator()
	if biometric.IsAvailable() {
		if err := biometric.Authenticate(ctx, "Unlock OpenPass vault"); err == nil {
			return LoadPassphrase(vaultDir)
		}
	}
	return "", ErrBiometricNotAvailable
}
