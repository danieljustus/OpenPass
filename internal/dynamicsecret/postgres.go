package dynamicsecret

import (
	"context"
	"errors"
)

// PostgreSQLEngine is a stub implementation of SecretEngine for PostgreSQL credentials.
// Wave 1: interface methods compile but do not connect to a real database.
type PostgreSQLEngine struct {
	connectionString string
}

// NewPostgreSQLEngine creates a new PostgreSQL engine stub.
func NewPostgreSQLEngine(connStr string) *PostgreSQLEngine {
	return &PostgreSQLEngine{connectionString: connStr}
}

// Type returns the engine type identifier.
func (e *PostgreSQLEngine) Type() string {
	return EngineTypePostgres
}

// Generate returns an error indicating the engine is not yet implemented.
func (e *PostgreSQLEngine) Generate(ctx context.Context, req GenerateRequest) (*Secret, error) {
	return nil, errors.New("postgres engine not implemented")
}

// Revoke returns an error indicating the engine is not yet implemented.
func (e *PostgreSQLEngine) Revoke(ctx context.Context, leaseID string) error {
	return errors.New("postgres engine not implemented")
}

// Validate returns an error indicating the engine is not yet implemented.
func (e *PostgreSQLEngine) Validate(ctx context.Context, req GenerateRequest) error {
	return errors.New("postgres engine not implemented")
}
