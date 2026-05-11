//go:build !dynamic_secrets

package dynamicsecret

import (
	"context"
	"errors"
)

// PostgreSQLEngine is a stub that returns an error when dynamic secrets are not compiled in.
type PostgreSQLEngine struct{}

// NewPostgreSQLEngine creates a stub PostgreSQL engine.
func NewPostgreSQLEngine(_ string) *PostgreSQLEngine {
	return &PostgreSQLEngine{}
}

// Type returns the engine type identifier.
func (e *PostgreSQLEngine) Type() string {
	return EngineTypePostgres
}

// Generate returns an error indicating dynamic secrets support is not compiled in.
func (e *PostgreSQLEngine) Generate(_ context.Context, _ GenerateRequest) (*Secret, error) {
	return nil, errors.New("dynamic secrets not compiled in (build with -tags dynamic_secrets)")
}

// Revoke returns an error indicating dynamic secrets support is not compiled in.
func (e *PostgreSQLEngine) Revoke(_ context.Context, _ string) error {
	return errors.New("dynamic secrets not compiled in (build with -tags dynamic_secrets)")
}

// Validate returns an error indicating dynamic secrets support is not compiled in.
func (e *PostgreSQLEngine) Validate(_ context.Context, _ GenerateRequest) error {
	return errors.New("dynamic secrets not compiled in (build with -tags dynamic_secrets)")
}
