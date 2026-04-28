package crypto

import (
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"math/big"
	"unicode"
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

// ValidatePasswordStrength checks if a password meets minimum strength requirements.
// It requires at least 10 characters and 60 bits of entropy based on charset diversity.
func ValidatePasswordStrength(password string) error {
	if len(password) < 10 {
		return fmt.Errorf("password too short: must be at least 10 characters")
	}

	charsetSize := 0
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSymbol := false

	for _, r := range password {
		if unicode.IsLower(r) {
			hasLower = true
		} else if unicode.IsUpper(r) {
			hasUpper = true
		} else if unicode.IsDigit(r) {
			hasDigit = true
		} else if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			hasSymbol = true
		}
	}

	if hasLower {
		charsetSize += 26
	}
	if hasUpper {
		charsetSize += 26
	}
	if hasDigit {
		charsetSize += 10
	}
	if hasSymbol {
		charsetSize += 32
	}
	if charsetSize == 0 {
		charsetSize = 256
	}

	entropy := float64(len(password)) * math.Log2(float64(charsetSize))
	if entropy < 60 {
		return fmt.Errorf("password too weak: estimated entropy %.1f bits, need at least 60 bits", entropy)
	}

	return nil
}
