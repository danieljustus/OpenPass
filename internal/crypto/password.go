package crypto

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
)

// MaxPasswordLength is the upper bound for generated password length.
const MaxPasswordLength = 1024

func GeneratePassword(length int, useSymbols bool) (string, error) {
	return generatePasswordWithReader(length, useSymbols, rand.Reader)
}

func generatePasswordWithReader(length int, useSymbols bool, reader io.Reader) (string, error) {
	const (
		letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		symbols = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	if length <= 0 {
		length = 16
	}
	if length > MaxPasswordLength {
		return "", fmt.Errorf("password length must be at most %d", MaxPasswordLength)
	}

	charset := letters
	if useSymbols {
		charset += symbols
	}

	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("generate password: %w", err)
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}
