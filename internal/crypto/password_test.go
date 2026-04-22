package crypto

import (
	"fmt"
	"strings"
	"testing"
)

func TestGeneratePassword_Length(t *testing.T) {
	for _, length := range []int{1, 8, 16, 32, 64} {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			password, err := GeneratePassword(length, true)
			if err != nil {
				t.Fatalf("GeneratePassword() error = %v", err)
			}
			if len(password) != length {
				t.Errorf("password length = %d, want %d", len(password), length)
			}
		})
	}
}

func TestGeneratePassword_ZeroLengthDefaultsTo16(t *testing.T) {
	password, err := GeneratePassword(0, true)
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	if len(password) != 16 {
		t.Errorf("password length = %d, want 16 (default)", len(password))
	}
}

func TestGeneratePassword_NegativeLengthDefaultsTo16(t *testing.T) {
	password, err := GeneratePassword(-5, true)
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	if len(password) != 16 {
		t.Errorf("password length = %d, want 16 (default)", len(password))
	}
}

func TestGeneratePassword_WithSymbols(t *testing.T) {
	password, err := GeneratePassword(50, true)
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}

	hasSymbol := false
	for _, c := range password {
		if strings.Contains("!@#$%^&*()_+-=[]{}|;:,.<>?", string(c)) {
			hasSymbol = true
			break
		}
	}
	if !hasSymbol {
		t.Error("expected password to contain at least one symbol")
	}
}

func TestGeneratePassword_WithoutSymbols(t *testing.T) {
	password, err := GeneratePassword(50, false)
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}

	for _, c := range password {
		if strings.Contains("!@#$%^&*()_+-=[]{}|;:,.<>?", string(c)) {
			t.Error("expected password to NOT contain symbols")
			break
		}
	}
}

func TestGeneratePassword_Randomness(t *testing.T) {
	password1, err := GeneratePassword(16, true)
	if err != nil {
		t.Fatalf("GeneratePassword() first call error = %v", err)
	}
	password2, err := GeneratePassword(16, true)
	if err != nil {
		t.Fatalf("GeneratePassword() second call error = %v", err)
	}
	if password1 == password2 {
		t.Error("expected different passwords on consecutive calls")
	}
}

func TestMaxPasswordLength(t *testing.T) {
	if MaxPasswordLength != 1024 {
		t.Errorf("MaxPasswordLength = %d, want 1024", MaxPasswordLength)
	}
}

func TestGeneratePassword_AtMaxLength(t *testing.T) {
	password, err := GeneratePassword(MaxPasswordLength, false)
	if err != nil {
		t.Fatalf("GeneratePassword(MaxPasswordLength) unexpected error: %v", err)
	}
	if len(password) != MaxPasswordLength {
		t.Errorf("password length = %d, want %d", len(password), MaxPasswordLength)
	}
}

func TestGeneratePassword_OverMaxLength(t *testing.T) {
	_, err := GeneratePassword(MaxPasswordLength+1, false)
	if err == nil {
		t.Fatal("GeneratePassword() error = nil, want error for length over maximum")
	}
}

func TestGeneratePassword_ErrorPath(t *testing.T) {
	failingReader := &errorReader{}
	_, err := generatePasswordWithReader(16, true, failingReader)
	if err == nil {
		t.Fatal("expected error from failing reader, got nil")
	}
	if !strings.Contains(err.Error(), "generate password:") {
		t.Errorf("expected error to contain 'generate password:', got %v", err)
	}
}

type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("mock reader error")
}
