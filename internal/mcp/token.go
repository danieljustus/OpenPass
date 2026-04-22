package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danieljustus/OpenPass/internal/vault"
)

var randReader = rand.Reader

func LoadOrCreateToken(path string) (string, error) { //nosec:G304
	data, err := os.ReadFile(path)
	if err == nil {
		token := strings.TrimSpace(string(data))
		if token != "" {
			if envToken := os.Getenv("OPENPASS_MCP_TOKEN"); envToken != "" {
				fmt.Fprintf(os.Stderr, "Warning: OPENPASS_MCP_TOKEN is set but file token exists at %s; using file token\n", path)
			}
			return token, nil
		}
	}

	if envToken := os.Getenv("OPENPASS_MCP_TOKEN"); envToken != "" {
		return envToken, nil
	}

	buf := make([]byte, 32)
	if _, err := randReader.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(buf)

	if err := vault.SafeWriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write token file: %w", err)
	}

	return token, nil
}

// RotateToken generates a new token and writes it to the token file.
// This invalidates the previous token - any MCP clients using the old token
// will need to be updated with the new token.
func RotateToken(path string) (string, error) {
	buf := make([]byte, 32)
	if _, err := randReader.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(buf)

	if err := vault.SafeWriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write token file: %w", err)
	}

	return token, nil
}

// TokenFilePath returns the default token file path for a vault directory.
func TokenFilePath(vaultDir string) string {
	return filepath.Join(vaultDir, "mcp-token")
}
