package crypto

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"

	"github.com/danieljustus/OpenPass/internal/pathutil"
)

var scryptWorkFactor = 18

// SetScryptWorkFactorForTests overrides the scrypt work factor for identities
// created in tests and returns a restore function.
func SetScryptWorkFactorForTests(workFactor int) func() {
	old := scryptWorkFactor
	scryptWorkFactor = workFactor
	return func() {
		scryptWorkFactor = old
	}
}

// GenerateIdentity generates a new age X25519 identity.
// Returns the generated identity or an error if generation fails.
func GenerateIdentity() (*age.X25519Identity, error) {
	return age.GenerateX25519Identity()
}

// validateIdentityPath ensures the identity file path doesn't escape expected directories.
func validateIdentityPath(path string) error {
	if pathutil.HasTraversal(path) {
		return errors.New("identity file path escapes expected directory")
	}
	return nil
}

// GenerateIdentityString generates a new age identity and returns it as a string.
func GenerateIdentityString() (string, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", err
	}
	return identity.String(), nil
}

// SaveIdentity encrypts and saves an identity to a file using a passphrase.
// The identity is encrypted with scrypt before being written to disk.
// The file permissions are set to 0o600 (readable/writable by owner only).
func SaveIdentity(id *age.X25519Identity, path string, passphrase []byte) error {
	if id == nil {
		return ErrNilIdentity
	}

	if len(passphrase) == 0 {
		return errors.New("passphrase is empty")
	}

	if err := validateIdentityPath(path); err != nil {
		return err
	}

	recipient, err := age.NewScryptRecipient(string(passphrase))
	if err != nil {
		return fmt.Errorf("create scrypt recipient: %w", err)
	}
	recipient.SetWorkFactor(scryptWorkFactor)

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return fmt.Errorf("create encryptor: %w", err)
	}

	if _, err := w.Write([]byte(id.String())); err != nil {
		return fmt.Errorf("write identity: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close encryptor: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// LoadIdentity loads and decrypts an identity from a file using a passphrase.
// Returns the decrypted identity or an error if loading/decryption fails.
func LoadIdentity(path string, passphrase []byte) (*age.X25519Identity, error) {
	if len(passphrase) == 0 {
		return nil, errors.New("passphrase is empty")
	}

	if err := validateIdentityPath(path); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path) //#nosec G304 -- path validated by validateIdentityPath()
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return nil, fmt.Errorf("create scrypt identity: %w", err)
	}

	r, err := age.Decrypt(bytes.NewReader(raw), identity)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read decrypted data: %w", err)
	}
	defer Wipe(plaintext)

	parsed, err := age.ParseX25519Identity(strings.TrimSpace(string(plaintext)))
	if err != nil {
		return nil, fmt.Errorf("parse identity: %w", err)
	}

	return parsed, nil
}

// IdentityExists checks if an identity file exists at the given path.
func IdentityExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetRecipientFromIdentity extracts the public recipient from an identity.
func GetRecipientFromIdentity(identity *age.X25519Identity) (*age.X25519Recipient, error) {
	if identity == nil {
		return nil, ErrNilIdentity
	}
	return identity.Recipient(), nil
}
