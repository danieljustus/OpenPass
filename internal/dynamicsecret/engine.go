// Package dynamicsecret provides dynamic secret generation and lease management
// for time-limited credentials across multiple backends (PostgreSQL, AWS STS, etc.).
package dynamicsecret

import (
	"context"
	"sync"
	"time"
)

// SecretEngine defines the interface for dynamic secret generation backends.
// Each engine manages the lifecycle of credentials for a specific service.
type SecretEngine interface {
	// Type returns the engine type identifier (e.g., "postgres", "aws-sts").
	Type() string

	// Generate creates a new dynamic secret based on the request parameters.
	Generate(ctx context.Context, req GenerateRequest) (*Secret, error)

	// Revoke invalidates a previously generated secret by lease ID.
	Revoke(ctx context.Context, leaseID string) error

	// Validate checks whether the request parameters are valid for this engine.
	Validate(ctx context.Context, req GenerateRequest) error
}

// EngineRegistry manages registered secret engines by type.
type EngineRegistry struct {
	mu      sync.RWMutex
	engines map[string]SecretEngine
}

// NewEngineRegistry creates an empty engine registry.
func NewEngineRegistry() *EngineRegistry {
	return &EngineRegistry{
		engines: make(map[string]SecretEngine),
	}
}

// Register adds a secret engine to the registry.
func (r *EngineRegistry) Register(engine SecretEngine) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.engines[engine.Type()] = engine
}

// Get retrieves a secret engine by type.
func (r *EngineRegistry) Get(engineType string) (SecretEngine, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	engine, ok := r.engines[engineType]
	return engine, ok
}

// List returns all registered engine type identifiers.
func (r *EngineRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.engines))
	for t := range r.engines {
		types = append(types, t)
	}
	return types
}

// GenerateRequest holds parameters for dynamic secret generation.
type GenerateRequest struct {
	// Role is the role or user template to use for the generated secret.
	Role string

	// TTL is the time-to-live for the generated secret.
	TTL time.Duration

	// Permissions defines the access level or scope (engine-specific).
	Permissions string
}

// Secret represents a dynamically generated secret with lease metadata.
type Secret struct {
	// LeaseID is the unique identifier for this secret lease.
	LeaseID string

	// LeaseDuration is the maximum lifetime of this secret.
	LeaseDuration time.Duration

	// Renewable indicates whether the lease can be renewed.
	Renewable bool

	// Data contains the actual secret material (credentials, tokens, etc.).
	Data map[string]any

	// CreatedAt is the timestamp when the secret was generated.
	CreatedAt time.Time

	// EngineType identifies the backend that generated this secret.
	EngineType string
}

// EngineType constants for supported backends.
const (
	EngineTypePostgres = "postgres"
	EngineTypeAWSSTS   = "aws-sts"
	EngineTypeMock     = "mock"
)
