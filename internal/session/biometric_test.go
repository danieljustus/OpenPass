package session

import (
	"context"
	"errors"
	"testing"
)

type mockBiometricAuthenticator struct {
	authErr   error
	available bool
}

func (m *mockBiometricAuthenticator) Authenticate(ctx context.Context, reason string) error {
	return m.authErr
}

func (m *mockBiometricAuthenticator) IsAvailable() bool {
	return m.available
}

func TestDefaultBiometricAuthenticator_NoopOnNil(t *testing.T) {
	biometricAuthenticator = nil
	auth := DefaultBiometricAuthenticator()
	if auth.IsAvailable() {
		t.Error("expected noop authenticator to not be available")
	}
	err := auth.Authenticate(context.Background(), "test")
	if !errors.Is(err, ErrBiometricNotAvailable) {
		t.Errorf("expected ErrBiometricNotAvailable, got %v", err)
	}
}

func TestDefaultBiometricAuthenticator_CustomAuthenticator(t *testing.T) {
	mock := &mockBiometricAuthenticator{available: true, authErr: nil}
	biometricAuthenticator = mock
	defer func() { biometricAuthenticator = nil }()

	auth := DefaultBiometricAuthenticator()
	if !auth.IsAvailable() {
		t.Error("expected mock authenticator to be available")
	}
	if err := auth.Authenticate(context.Background(), "test"); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestSetBiometricAuthenticator(t *testing.T) {
	mock := &mockBiometricAuthenticator{available: true, authErr: nil}
	SetBiometricAuthenticator(mock)
	defer func() { biometricAuthenticator = nil }()

	auth := DefaultBiometricAuthenticator()
	if auth != mock {
		t.Error("expected SetBiometricAuthenticator to return the set authenticator")
	}
}

func TestLoadPassphraseWithTouchID_Available(t *testing.T) {
	mock := &mockBiometricAuthenticator{available: true, authErr: nil}
	biometricAuthenticator = mock
	defer func() { biometricAuthenticator = nil }()

	_, err := LoadPassphraseWithTouchID(context.Background(), "/nonexistent")
	if err == nil {
		t.Fatal("expected error for missing cached passphrase")
	}
}

func TestLoadPassphraseWithTouchID_NotAvailable(t *testing.T) {
	mock := &mockBiometricAuthenticator{available: false}
	biometricAuthenticator = mock
	defer func() { biometricAuthenticator = nil }()

	_, err := LoadPassphraseWithTouchID(context.Background(), "/nonexistent")
	if !errors.Is(err, ErrBiometricNotAvailable) {
		t.Errorf("expected ErrBiometricNotAvailable when not available, got %v", err)
	}
}

func TestLoadPassphraseWithTouchID_AuthFails(t *testing.T) {
	mock := &mockBiometricAuthenticator{available: true, authErr: ErrBiometricFailed}
	biometricAuthenticator = mock
	defer func() { biometricAuthenticator = nil }()

	_, err := LoadPassphraseWithTouchID(context.Background(), "/nonexistent")
	if !errors.Is(err, ErrBiometricNotAvailable) {
		t.Errorf("expected ErrBiometricNotAvailable when auth fails, got %v", err)
	}
}

func TestNoopBiometricAuthenticator(t *testing.T) {
	noop := noopBiometricAuthenticator{}
	if noop.IsAvailable() {
		t.Error("noop should not be available")
	}
	err := noop.Authenticate(context.Background(), "test")
	if !errors.Is(err, ErrBiometricNotAvailable) {
		t.Errorf("expected ErrBiometricNotAvailable, got %v", err)
	}
}
