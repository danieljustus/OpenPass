package dynamicsecret

import (
	"context"
	"time"

	"github.com/danieljustus/OpenPass/internal/vaultsvc"
)

// Manager orchestrates dynamic secret generation across multiple engines.
// It provides the high-level API for requesting, renewing, and revoking secrets.
type Manager struct {
	vault    vaultsvc.Service
	registry *EngineRegistry
	leases   *LeaseManager
}

// NewManager creates a new dynamic secret manager with the given vault service.
func NewManager(vault vaultsvc.Service) *Manager {
	return &Manager{
		vault:    vault,
		registry: NewEngineRegistry(),
		leases:   NewLeaseManager(),
	}
}

// Generate creates a new dynamic secret using the specified engine.
func (m *Manager) Generate(ctx context.Context, engineType string, req GenerateRequest) (*Secret, error) {
	return nil, nil
}

// Revoke invalidates a secret by lease ID.
func (m *Manager) Revoke(ctx context.Context, leaseID string) error {
	return nil
}

// Renew extends the TTL of an existing secret lease.
func (m *Manager) Renew(ctx context.Context, leaseID string, increment time.Duration) (*Secret, error) {
	return nil, nil
}

// Lookup retrieves a secret by lease ID.
func (m *Manager) Lookup(ctx context.Context, leaseID string) (*Secret, error) {
	return nil, nil
}

// RegisterEngine adds a secret engine to the manager's registry.
func (m *Manager) RegisterEngine(engine SecretEngine) {
	m.registry.Register(engine)
}

// ListEngines returns all registered engine types.
func (m *Manager) ListEngines() []string {
	return m.registry.List()
}

// Close shuts down the manager and its lease cleanup routines.
func (m *Manager) Close() error {
	return m.leases.Close()
}
