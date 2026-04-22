package cmd

import (
	"strings"
	"testing"

	"github.com/danieljustus/OpenPass/internal/crypto"
)

func TestGeneratePassword_ValidLengths(t *testing.T) {
	for _, length := range []int{1, 20, 100, 1024, crypto.MaxPasswordLength} {
		t.Run("", func(t *testing.T) {
			password, err := generatePassword(length, false)
			if err != nil {
				t.Fatalf("generatePassword(%d) unexpected error: %v", length, err)
			}
			if len(password) != length {
				t.Errorf("password length = %d, want %d", len(password), length)
			}
		})
	}
}

func TestGeneratePassword_ZeroLength(t *testing.T) {
	_, err := generatePassword(0, false)
	if err == nil {
		t.Fatal("expected error for length=0, got nil")
	}
	if !strings.Contains(err.Error(), "greater than zero") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGeneratePassword_NegativeLength(t *testing.T) {
	_, err := generatePassword(-1, false)
	if err == nil {
		t.Fatal("expected error for length=-1, got nil")
	}
	if !strings.Contains(err.Error(), "greater than zero") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGeneratePassword_ExceedsMaxLength(t *testing.T) {
	_, err := generatePassword(crypto.MaxPasswordLength+1, false)
	if err == nil {
		t.Fatalf("expected error for length=%d, got nil", crypto.MaxPasswordLength+1)
	}
	if !strings.Contains(err.Error(), "at most") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGeneratePassword_WithSymbols(t *testing.T) {
	password, err := generatePassword(50, true)
	if err != nil {
		t.Fatalf("generatePassword(50, true) error: %v", err)
	}
	if len(password) != 50 {
		t.Errorf("password length = %d, want 50", len(password))
	}
}
