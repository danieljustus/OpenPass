package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// GrantIDFields contains the fields used to compute the HMAC for a
// cryptographically bound share grant ID.
type GrantIDFields struct {
	FromAgent   string    `json:"from_agent"`
	ToAgent     string    `json:"to_agent"`
	SecretPath  string    `json:"secret_path"`
	SecretField string    `json:"secret_field,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Nonce       string    `json:"nonce"`
}

const (
	// NonceSize is the number of random bytes for a grant nonce.
	NonceSize = 16
)

// GenerateGrantID creates a cryptographically bound grant ID in the format
// "nonce_hex:hmac_hex". The HMAC is SHA256 over the canonical JSON encoding
// of fields using the provided key.
func GenerateGrantID(fields GrantIDFields, key []byte) (string, error) {
	if len(key) == 0 {
		return "", fmt.Errorf("hmac key is empty")
	}

	if fields.Nonce == "" {
		nonce := make([]byte, NonceSize)
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return "", fmt.Errorf("generate nonce: %w", err)
		}
		fields.Nonce = hex.EncodeToString(nonce)
	}

	canonical, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("marshal grant fields: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(canonical)
	hmacVal := hex.EncodeToString(mac.Sum(nil))

	return fields.Nonce + ":" + hmacVal, nil
}

// VerifyGrantID verifies that the given grant ID is valid for the specified
// fields and key. It uses constant-time comparison via hmac.Equal. Only
// HMAC-format IDs (containing ":") are verified; legacy UUID IDs are
// reported as non-HMAC format via IsHMACFormat.
func VerifyGrantID(grantID string, fields GrantIDFields, key []byte) (bool, error) {
	if len(key) == 0 {
		return false, fmt.Errorf("hmac key is empty")
	}

	if !IsHMACFormat(grantID) {
		return false, fmt.Errorf("grant ID %q is not in HMAC format", grantID)
	}

	parts := strings.SplitN(grantID, ":", 2)
	nonce := parts[0]
	expectedHMAC := parts[1]

	if _, err := hex.DecodeString(nonce); err != nil {
		return false, fmt.Errorf("invalid nonce in grant ID: %w", err)
	}

	expected, err := hex.DecodeString(expectedHMAC)
	if err != nil {
		return false, fmt.Errorf("invalid hmac in grant ID: %w", err)
	}

	fields.Nonce = nonce
	canonical, err := json.Marshal(fields)
	if err != nil {
		return false, fmt.Errorf("marshal grant fields: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(canonical)
	computed := mac.Sum(nil)

	return hmac.Equal(computed, expected), nil
}

// IsHMACFormat returns true if the grant ID uses the HMAC format
// (nonce_hex:hmac_hex) rather than a legacy UUID. HMAC-format IDs always
// contain a colon separator.
func IsHMACFormat(grantID string) bool {
	return strings.Contains(grantID, ":")
}

// ParseNonceFromID extracts the nonce portion from an HMAC-format grant ID.
// Returns an error if the ID is not in HMAC format.
func ParseNonceFromID(grantID string) (string, error) {
	if !IsHMACFormat(grantID) {
		return "", fmt.Errorf("grant ID %q is not in HMAC format", grantID)
	}
	parts := strings.SplitN(grantID, ":", 2)
	return parts[0], nil
}
