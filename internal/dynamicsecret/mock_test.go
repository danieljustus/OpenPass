package dynamicsecret

import (
	"context"
	"testing"
)

func TestMockEngineImplementsInterface(t *testing.T) {
	var _ SecretEngine = NewMockEngine()
}

func TestMockEngineTypeDefault(t *testing.T) {
	mock := NewMockEngine()
	if mock.Type() != EngineTypeMock {
		t.Errorf("Type() = %q, want %q", mock.Type(), EngineTypeMock)
	}
}

func TestMockEngineGenerateDefault(t *testing.T) {
	mock := NewMockEngine()
	secret, err := mock.Generate(context.Background(), GenerateRequest{})
	if err != nil {
		t.Errorf("Generate error: %v", err)
	}
	if secret != nil {
		t.Error("Generate should return nil by default")
	}
}

func TestMockEngineRevokeDefault(t *testing.T) {
	mock := NewMockEngine()
	if err := mock.Revoke(context.Background(), "any"); err != nil {
		t.Errorf("Revoke error: %v", err)
	}
}

func TestMockEngineValidateDefault(t *testing.T) {
	mock := NewMockEngine()
	if err := mock.Validate(context.Background(), GenerateRequest{}); err != nil {
		t.Errorf("Validate error: %v", err)
	}
}

func TestMockEngineCustomGenerate(t *testing.T) {
	mock := NewMockEngine()
	expected := &Secret{LeaseID: "custom-lease", EngineType: EngineTypeMock}

	mock.GenerateFunc = func(ctx context.Context, req GenerateRequest) (*Secret, error) {
		return expected, nil
	}

	got, err := mock.Generate(context.Background(), GenerateRequest{Role: "test"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if got != expected {
		t.Error("Generate did not return custom secret")
	}
}

func TestMockEngineCustomRevoke(t *testing.T) {
	mock := NewMockEngine()
	mock.RevokeFunc = func(ctx context.Context, leaseID string) error {
		return context.Canceled
	}

	err := mock.Revoke(context.Background(), "lease-1")
	if err != context.Canceled {
		t.Errorf("Revoke error = %v, want context.Canceled", err)
	}
}

func TestMockEngineCustomValidate(t *testing.T) {
	mock := NewMockEngine()
	mock.ValidateFunc = func(ctx context.Context, req GenerateRequest) error {
		return context.DeadlineExceeded
	}

	err := mock.Validate(context.Background(), GenerateRequest{})
	if err != context.DeadlineExceeded {
		t.Errorf("Validate error = %v, want context.DeadlineExceeded", err)
	}
}

func TestMockEngineCustomType(t *testing.T) {
	mock := NewMockEngine()
	mock.TypeFunc = func() string {
		return "custom-mock"
	}

	if mock.Type() != "custom-mock" {
		t.Errorf("Type() = %q, want custom-mock", mock.Type())
	}
}

func TestMockEngineAllMethodsCallable(t *testing.T) {
	mock := NewMockEngine()

	_ = mock.Type()
	_, _ = mock.Generate(context.Background(), GenerateRequest{})
	_ = mock.Revoke(context.Background(), "lease-1")
	_ = mock.Validate(context.Background(), GenerateRequest{})
}
