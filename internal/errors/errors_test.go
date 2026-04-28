package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewCLIError(t *testing.T) {
	err := NewCLIError(ExitLocked, "vault locked", errors.New("passphrase missing"))
	if err.Code != ExitLocked {
		t.Errorf("expected code %d, got %d", ExitLocked, err.Code)
	}
	if err.Message != "vault locked" {
		t.Errorf("expected message %q, got %q", "vault locked", err.Message)
	}
	if err.Cause == nil || err.Cause.Error() != "passphrase missing" {
		t.Error("expected cause to match")
	}
}

func TestCLIError_Error(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		err := NewCLIError(ExitGeneralError, "cannot list", fmt.Errorf("io error"))
		want := "cannot list: io error"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without cause", func(t *testing.T) {
		err := NewCLIError(ExitNotFound, "invalid argument", nil)
		want := "invalid argument"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestExitCodeFromError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ExitCode
	}{
		{"nil error", nil, ExitSuccess},
		{"plain error", fmt.Errorf("plain"), ExitGeneralError},
		{"CLIError vault locked", NewCLIError(ExitLocked, "locked", nil), ExitLocked},
		{"wrapped CLIError", fmt.Errorf("outer: %w", NewCLIError(ExitNotInitialized, "not init", nil)), ExitNotInitialized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCodeFromError(tt.err); got != tt.want {
				t.Errorf("ExitCodeFromError() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if err := Wrap(ExitGeneralError, "msg", nil); err != nil {
			t.Errorf("Wrap(nil) = %v, want nil", err)
		}
	})

	t.Run("wraps error", func(t *testing.T) {
		inner := fmt.Errorf("inner")
		err := Wrap(ExitPermissionDenied, "server failed", inner)
		var cliErr *CLIError
		if !errors.As(err, &cliErr) {
			t.Fatal("expected *CLIError")
		}
		if cliErr.Code != ExitPermissionDenied {
			t.Errorf("code = %d, want %d", cliErr.Code, ExitPermissionDenied)
		}
		if !errors.Is(err, inner) {
			t.Error("expected wrapped error to be unwrappable")
		}
	})
}
