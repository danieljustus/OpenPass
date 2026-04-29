package session

import (
	"context"
	"errors"
)

var ErrBiometricNotAvailable = errors.New("biometric authentication not available")
var ErrBiometricFailed = errors.New("biometric authentication failed")
var ErrBiometricNotConfigured = errors.New("biometric authentication is not configured")

type BiometricAuthenticator interface {
	Authenticate(ctx context.Context, reason string) error
	IsAvailable() bool
}

type BiometricPassphraseStore interface {
	IsAvailable() bool
	Save(ctx context.Context, vaultDir string, passphrase string) error
	Load(ctx context.Context, vaultDir string) (string, error)
	Delete(vaultDir string) error
}

var biometricAuthenticator BiometricAuthenticator
var biometricPassphraseStore BiometricPassphraseStore

func DefaultBiometricAuthenticator() BiometricAuthenticator {
	if biometricAuthenticator != nil {
		return biometricAuthenticator
	}
	return noopBiometricAuthenticator{}
}

func SetBiometricAuthenticator(a BiometricAuthenticator) {
	biometricAuthenticator = a
}

func DefaultBiometricPassphraseStore() BiometricPassphraseStore {
	if biometricPassphraseStore != nil {
		return biometricPassphraseStore
	}
	return noopBiometricPassphraseStore{}
}

func SetBiometricPassphraseStore(store BiometricPassphraseStore) {
	biometricPassphraseStore = store
}

func BiometricAvailable() bool {
	return DefaultBiometricPassphraseStore().IsAvailable()
}

func SaveBiometricPassphrase(ctx context.Context, vaultDir string, passphrase string) error {
	return DefaultBiometricPassphraseStore().Save(ctx, vaultDir, passphrase)
}

func LoadBiometricPassphrase(ctx context.Context, vaultDir string) (string, error) {
	return DefaultBiometricPassphraseStore().Load(ctx, vaultDir)
}

func ClearBiometricPassphrase(vaultDir string) error {
	return DefaultBiometricPassphraseStore().Delete(vaultDir)
}

type noopBiometricAuthenticator struct{}

func (noopBiometricAuthenticator) Authenticate(ctx context.Context, reason string) error {
	return ErrBiometricNotAvailable
}

func (noopBiometricAuthenticator) IsAvailable() bool {
	return false
}

type noopBiometricPassphraseStore struct{}

func (noopBiometricPassphraseStore) IsAvailable() bool {
	return false
}

func (noopBiometricPassphraseStore) Save(_ context.Context, _ string, _ string) error {
	return ErrBiometricNotAvailable
}

func (noopBiometricPassphraseStore) Load(ctx context.Context, vaultDir string) (string, error) {
	_, _ = ctx, vaultDir
	return "", ErrBiometricNotAvailable
}

func (noopBiometricPassphraseStore) Delete(vaultDir string) error {
	_ = vaultDir
	return nil
}
