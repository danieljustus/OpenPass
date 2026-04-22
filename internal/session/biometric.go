package session

import (
	"context"
	"errors"
)

var ErrBiometricNotAvailable = errors.New("biometric authentication not available")
var ErrBiometricFailed = errors.New("biometric authentication failed")

type BiometricAuthenticator interface {
	Authenticate(ctx context.Context, reason string) error
	IsAvailable() bool
}

var biometricAuthenticator BiometricAuthenticator

func DefaultBiometricAuthenticator() BiometricAuthenticator {
	if biometricAuthenticator != nil {
		return biometricAuthenticator
	}
	return noopBiometricAuthenticator{}
}

func SetBiometricAuthenticator(a BiometricAuthenticator) {
	biometricAuthenticator = a
}

type noopBiometricAuthenticator struct{}

func (noopBiometricAuthenticator) Authenticate(ctx context.Context, reason string) error {
	return ErrBiometricNotAvailable
}

func (noopBiometricAuthenticator) IsAvailable() bool {
	return false
}
