package crypto

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateTOTP_Defaults(t *testing.T) {
	// Base32-encoded "12345678901234567890"
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	code, err := GenerateTOTP(secret, "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code.Code) != 6 {
		t.Errorf("expected 6-digit code, got %d digits: %s", len(code.Code), code.Code)
	}
	if code.Period != 30 {
		t.Errorf("expected period 30, got %d", code.Period)
	}
	if code.ExpiresAt.Before(time.Now()) {
		t.Error("expiration should be in the future")
	}
}

func TestGenerateTOTP_CustomDigits(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	code, err := GenerateTOTP(secret, "SHA1", 8, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code.Code) != 8 {
		t.Errorf("expected 8-digit code, got %d digits: %s", len(code.Code), code.Code)
	}
}

func TestGenerateTOTP_SHA256(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	code, err := GenerateTOTP(secret, "SHA256", 6, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code.Code) != 6 {
		t.Errorf("expected 6-digit code, got %s", code.Code)
	}
}

func TestGenerateTOTP_SHA512(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	code, err := GenerateTOTP(secret, "SHA512", 6, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code.Code) != 6 {
		t.Errorf("expected 6-digit code, got %s", code.Code)
	}
}

func TestGenerateTOTP_InvalidSecret(t *testing.T) {
	_, err := GenerateTOTP("not-valid-base32!!!", "", 0, 0)
	if err == nil {
		t.Error("expected error for invalid secret")
	}
}

func TestGenerateTOTP_SecretWithSpaces(t *testing.T) {
	// Many authenticator apps display secrets with spaces
	secret := "GEZD GNBV GY3T QOJQ GEZD GNBV GY3T QOJQ"

	code, err := GenerateTOTP(secret, "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code.Code) != 6 {
		t.Errorf("expected 6-digit code, got %s", code.Code)
	}
}

func TestGenerateTOTP_LowercaseSecret(t *testing.T) {
	secret := "gezdgnbvgy3tqojqgezdgnbvgy3tqojq"

	code, err := GenerateTOTP(secret, "", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code.Code) != 6 {
		t.Errorf("expected 6-digit code, got %s", code.Code)
	}
}

func TestGenerateTOTP_CustomPeriod(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	code, err := GenerateTOTP(secret, "", 6, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code.Period != 60 {
		t.Errorf("expected period 60, got %d", code.Period)
	}
}

func TestGenerateTOTP_Deterministic(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	// Two calls within the same time step should return the same code
	code1, err := GenerateTOTP(secret, "SHA1", 6, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	code2, err := GenerateTOTP(secret, "SHA1", 6, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code1.Code != code2.Code {
		t.Errorf("codes should match within same time step: %s vs %s", code1.Code, code2.Code)
	}
}

func TestGenerateTOTP_DifferentAlgorithmsDifferentCodes(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	sha1Code, _ := GenerateTOTP(secret, "SHA1", 6, 30)
	sha256Code, _ := GenerateTOTP(secret, "SHA256", 6, 30)
	sha512Code, _ := GenerateTOTP(secret, "SHA512", 6, 30)

	// At least two of the three should differ (extremely unlikely all match)
	if sha1Code.Code == sha256Code.Code && sha256Code.Code == sha512Code.Code {
		t.Error("all three algorithms produced the same code — extremely unlikely")
	}
}

func TestValidateTOTPSecret_ValidPadded(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if err := ValidateTOTPSecret(secret); err != nil {
		t.Errorf("ValidateTOTPSecret(%q) = %v; want nil", secret, err)
	}
}

func TestValidateTOTPSecret_ValidUnpadded(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	if err := ValidateTOTPSecret(secret); err != nil {
		t.Errorf("ValidateTOTPSecret(%q) = %v; want nil", secret, err)
	}
}

func TestValidateTOTPSecret_WithSpaces(t *testing.T) {
	secret := "JBSW Y3DP EHPK 3PXP"
	if err := ValidateTOTPSecret(secret); err != nil {
		t.Errorf("ValidateTOTPSecret(%q) = %v; want nil", secret, err)
	}
}

func TestValidateTOTPSecret_Lowercase(t *testing.T) {
	secret := "jbswy3dpehpk3pxp"
	if err := ValidateTOTPSecret(secret); err != nil {
		t.Errorf("ValidateTOTPSecret(%q) = %v; want nil", secret, err)
	}
}

func TestValidateTOTPSecret_InvalidRejected(t *testing.T) {
	secret := "not-valid-base32!!!"
	err := ValidateTOTPSecret(secret)
	if err == nil {
		t.Fatal("expected error for invalid secret")
	}
	want := "TOTP secret must be Base32-encoded (spaces allowed)"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	if strings.Contains(err.Error(), secret) {
		t.Error("error message must not contain the secret value")
	}
}

func TestValidateTOTPParams_ValidCases(t *testing.T) {
	validCases := []struct {
		algo   string
		digits int
		period int
	}{
		{"SHA1", 6, 30},
		{"SHA256", 6, 30},
		{"SHA512", 8, 60},
		{"sha1", 8, 3600},
		{"", 6, 1},
		{"", 0, 0},
	}
	for _, tc := range validCases {
		if err := ValidateTOTPParams(tc.algo, tc.digits, tc.period); err != nil {
			t.Errorf("ValidateTOTPParams(%q, %d, %d) = %v; want nil", tc.algo, tc.digits, tc.period, err)
		}
	}
}

func TestValidateTOTPParams_InvalidAlgorithm(t *testing.T) {
	if err := ValidateTOTPParams("MD5", 6, 30); err == nil {
		t.Error("expected error for invalid algorithm MD5")
	}
}

func TestValidateTOTPParams_InvalidDigits(t *testing.T) {
	if err := ValidateTOTPParams("SHA1", 7, 30); err == nil {
		t.Error("expected error for invalid digits 7")
	}
	if err := ValidateTOTPParams("SHA1", 5, 30); err == nil {
		t.Error("expected error for invalid digits 5")
	}
	// zero is valid - means "unset" and will be replaced with defaults
	if err := ValidateTOTPParams("SHA1", 0, 30); err != nil {
		t.Errorf("zero digits should be valid (unset): %v", err)
	}
}

func TestValidateTOTPParams_InvalidPeriod(t *testing.T) {
	// zero is valid - means "unset" and will be replaced with defaults
	if err := ValidateTOTPParams("SHA1", 6, 0); err != nil {
		t.Errorf("zero period should be valid (unset): %v", err)
	}
	if err := ValidateTOTPParams("SHA1", 6, -1); err == nil {
		t.Error("expected error for negative period")
	}
	if err := ValidateTOTPParams("SHA1", 6, 3601); err == nil {
		t.Error("expected error for period > 3600")
	}
}
