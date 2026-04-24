//go:build darwin

package session

import (
	"context"
	"testing"
)

func TestTouchIDAuthenticator_IsAvailable(t *testing.T) {
	auth := &touchIDAuthenticator{}
	_ = auth.IsAvailable()
}

func TestTouchIDAuthenticator_Authenticate(t *testing.T) {
	auth := &touchIDAuthenticator{}
	err := auth.Authenticate(context.Background(), "test")
	if err == nil {
		t.Log("TouchID auth succeeded (unexpected in test environment)")
	}
}

func TestTouchIDAvailable_Function(t *testing.T) {
	result := touchIDAvailable()
	t.Logf("touchIDAvailable() = %v", result)
}

func TestTouchIDAuthenticate_Function(t *testing.T) {
	result := touchIDAuthenticate(context.Background(), "test reason")
	t.Logf("touchIDAuthenticate() = %v", result)
}
