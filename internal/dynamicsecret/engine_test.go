package dynamicsecret

import (
	"context"
	"testing"
	"time"
)

func TestSecretEngineInterface(t *testing.T) {
	// Verify MockEngine implements SecretEngine
	var _ SecretEngine = NewMockEngine()
}

func TestGenerateRequestFields(t *testing.T) {
	req := GenerateRequest{
		Role:        "admin",
		TTL:         time.Hour,
		Permissions: "read-write",
	}

	if req.Role != "admin" {
		t.Errorf("Role = %q, want admin", req.Role)
	}
	if req.TTL != time.Hour {
		t.Errorf("TTL = %v, want 1h", req.TTL)
	}
	if req.Permissions != "read-write" {
		t.Errorf("Permissions = %q, want read-write", req.Permissions)
	}
}

func TestSecretFields(t *testing.T) {
	now := time.Now().UTC()
	secret := Secret{
		LeaseID:       "lease-123",
		LeaseDuration: time.Hour,
		Renewable:     true,
		Data:          map[string]any{"password": "secret"},
		CreatedAt:     now,
		EngineType:    EngineTypeMock,
	}

	if secret.LeaseID != "lease-123" {
		t.Errorf("LeaseID = %q, want lease-123", secret.LeaseID)
	}
	if secret.LeaseDuration != time.Hour {
		t.Errorf("LeaseDuration = %v, want 1h", secret.LeaseDuration)
	}
	if !secret.Renewable {
		t.Error("Renewable = false, want true")
	}
	if secret.Data["password"] != "secret" {
		t.Errorf("Data[password] = %v, want secret", secret.Data["password"])
	}
	if !secret.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch")
	}
	if secret.EngineType != EngineTypeMock {
		t.Errorf("EngineType = %q, want mock", secret.EngineType)
	}
}

func TestEngineRegistry(t *testing.T) {
	reg := NewEngineRegistry()

	// Test empty registry
	if engines := reg.List(); len(engines) != 0 {
		t.Fatalf("List() = %v, want empty", engines)
	}

	// Test register and get
	mock := NewMockEngine()
	reg.Register(mock)

	engines := reg.List()
	if len(engines) != 1 {
		t.Fatalf("List() = %v, want 1 engine", engines)
	}

	got, ok := reg.Get(EngineTypeMock)
	if !ok {
		t.Fatal("Get(mock) = false, want true")
	}
	if got.Type() != EngineTypeMock {
		t.Errorf("Type() = %q, want mock", got.Type())
	}

	// Test get missing
	_, ok = reg.Get("missing")
	if ok {
		t.Error("Get(missing) = true, want false")
	}
}

func TestEngineRegistryConcurrent(t *testing.T) {
	reg := NewEngineRegistry()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			reg.Register(NewMockEngine())
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		go func() {
			reg.Get(EngineTypeMock)
			done <- struct{}{}
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestEngineTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"postgres", EngineTypePostgres, "postgres"},
		{"aws-sts", EngineTypeAWSSTS, "aws-sts"},
		{"mock", EngineTypeMock, "mock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("EngineType = %q, want %q", tt.value, tt.want)
			}
		})
	}
}

func TestMockEngineDefaults(t *testing.T) {
	mock := NewMockEngine()

	if mock.Type() != EngineTypeMock {
		t.Errorf("Type() = %q, want %q", mock.Type(), EngineTypeMock)
	}

	secret, err := mock.Generate(context.Background(), GenerateRequest{})
	if err != nil {
		t.Errorf("Generate error: %v", err)
	}
	if secret != nil {
		t.Error("Generate should return nil by default")
	}

	if err := mock.Revoke(context.Background(), "any"); err != nil {
		t.Errorf("Revoke error: %v", err)
	}

	if err := mock.Validate(context.Background(), GenerateRequest{}); err != nil {
		t.Errorf("Validate error: %v", err)
	}
}

func TestMockEngineCustomBehavior(t *testing.T) {
	mock := NewMockEngine()

	customSecret := &Secret{LeaseID: "custom-lease"}
	mock.GenerateFunc = func(ctx context.Context, req GenerateRequest) (*Secret, error) {
		return customSecret, nil
	}

	got, err := mock.Generate(context.Background(), GenerateRequest{Role: "admin"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if got != customSecret {
		t.Error("Generate did not return custom secret")
	}

	mock.RevokeFunc = func(ctx context.Context, leaseID string) error {
		return context.Canceled
	}
	if err := mock.Revoke(context.Background(), "lease-1"); err != context.Canceled {
		t.Errorf("Revoke error = %v, want context.Canceled", err)
	}
}

func TestMockEngineWithRequest(t *testing.T) {
	mock := NewMockEngine()

	var capturedReq GenerateRequest
	mock.GenerateFunc = func(ctx context.Context, req GenerateRequest) (*Secret, error) {
		capturedReq = req
		return &Secret{LeaseID: "lease-1"}, nil
	}

	req := GenerateRequest{
		Role:        "reader",
		TTL:         30 * time.Minute,
		Permissions: "read-only",
	}
	secret, err := mock.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if secret == nil {
		t.Fatal("Generate returned nil secret")
	}
	if capturedReq.Role != "reader" {
		t.Errorf("captured Role = %q, want reader", capturedReq.Role)
	}
	if capturedReq.TTL != 30*time.Minute {
		t.Errorf("captured TTL = %v, want 30m", capturedReq.TTL)
	}
	if capturedReq.Permissions != "read-only" {
		t.Errorf("captured Permissions = %q, want read-only", capturedReq.Permissions)
	}
}
