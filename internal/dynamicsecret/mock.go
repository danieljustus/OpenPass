package dynamicsecret

import "context"

// MockEngine provides a mock implementation of SecretEngine for testing.
// Set any Func field to customize behavior; defaults return zero values.
type MockEngine struct {
	TypeFunc     func() string
	GenerateFunc func(ctx context.Context, req GenerateRequest) (*Secret, error)
	RevokeFunc   func(ctx context.Context, leaseID string) error
	ValidateFunc func(ctx context.Context, req GenerateRequest) error
}

// NewMockEngine creates a MockEngine with sensible no-op defaults.
func NewMockEngine() *MockEngine {
	return &MockEngine{
		TypeFunc: func() string {
			return EngineTypeMock
		},
		GenerateFunc: func(ctx context.Context, req GenerateRequest) (*Secret, error) {
			return nil, nil
		},
		RevokeFunc: func(ctx context.Context, leaseID string) error {
			return nil
		},
		ValidateFunc: func(ctx context.Context, req GenerateRequest) error {
			return nil
		},
	}
}

func (m *MockEngine) Type() string {
	return m.TypeFunc()
}

func (m *MockEngine) Generate(ctx context.Context, req GenerateRequest) (*Secret, error) {
	return m.GenerateFunc(ctx, req)
}

func (m *MockEngine) Revoke(ctx context.Context, leaseID string) error {
	return m.RevokeFunc(ctx, leaseID)
}

func (m *MockEngine) Validate(ctx context.Context, req GenerateRequest) error {
	return m.ValidateFunc(ctx, req)
}
