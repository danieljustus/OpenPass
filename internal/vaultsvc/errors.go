package vaultsvc

import (
	"errors"
	"fmt"
)

// ErrorKind categorizes vault service errors for consistent mapping.
type ErrorKind int

const (
	ErrNotFound ErrorKind = iota
	ErrFieldNotFound
	ErrReadFailed
	ErrWriteFailed
	ErrVaultNotInitialized
	ErrVaultLocked
	ErrPermissionDenied
)

// Error is a domain error from the vault service layer.
type Error struct {
	Kind    ErrorKind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// IsNotFound returns true if the error or any wrapped error is a not-found error.
func IsNotFound(err error) bool {
	var svcErr *Error
	if errors.As(err, &svcErr) {
		return svcErr.Kind == ErrNotFound || svcErr.Kind == ErrFieldNotFound
	}
	return false
}

// IsWriteError returns true if the error is a write failure.
func IsWriteError(err error) bool {
	var svcErr *Error
	if errors.As(err, &svcErr) {
		return svcErr.Kind == ErrWriteFailed
	}
	return false
}

// NewError creates a new vault service error.
func NewError(kind ErrorKind, message string, cause error) *Error {
	return &Error{Kind: kind, Message: message, Cause: cause}
}
