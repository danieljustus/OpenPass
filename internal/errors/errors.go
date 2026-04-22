// Package errors provides centralized error handling for OpenPass CLI commands.
// It defines typed errors with exit codes and consistent error wrapping.
package errors

import (
	"errors"
	"fmt"
)

// ExitCode represents a process exit code for CLI commands.
type ExitCode int

const (
	// ExitSuccess indicates successful completion.
	ExitSuccess ExitCode = 0
	// ExitGeneralError indicates a general error.
	ExitGeneralError ExitCode = 1
	// ExitUsageError indicates a command-line usage error.
	ExitUsageError ExitCode = 2
	// ExitVaultLocked indicates the vault is locked or passphrase is missing.
	ExitVaultLocked ExitCode = 3
	// ExitNotInitialized indicates the vault is not initialized.
	ExitNotInitialized ExitCode = 4
	// ExitMCPError indicates an MCP server error.
	ExitMCPError ExitCode = 5
)

// CLIError is a structured error with an exit code and user-friendly message.
type CLIError struct {
	Code    ExitCode
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *CLIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause for errors.Is/errors.As support.
func (e *CLIError) Unwrap() error {
	return e.Cause
}

// NewCLIError creates a new CLIError with the given code, message, and optional cause.
func NewCLIError(code ExitCode, msg string, cause error) *CLIError {
	return &CLIError{
		Code:    code,
		Message: msg,
		Cause:   cause,
	}
}

// ExitCodeFromError extracts the exit code from an error.
// If the error is a *CLIError, it returns the embedded code.
// Otherwise, it returns ExitGeneralError.
func ExitCodeFromError(err error) ExitCode {
	if err == nil {
		return ExitSuccess
	}
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr.Code
	}
	return ExitGeneralError
}

// Wrap wraps an existing error with a message and exit code.
func Wrap(code ExitCode, msg string, err error) error {
	if err == nil {
		return nil
	}
	return NewCLIError(code, msg, err)
}
