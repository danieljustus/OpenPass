package dynamicsecret

import (
	"context"
	"strings"
	"testing"
)

func TestPostgreSQLEngineImplementsInterface(t *testing.T) {
	var _ SecretEngine = NewPostgreSQLEngine("postgres://localhost")
}

func TestPostgreSQLEngineType(t *testing.T) {
	engine := NewPostgreSQLEngine("postgres://localhost")
	if engine.Type() != EngineTypePostgres {
		t.Errorf("Type() = %q, want %q", engine.Type(), EngineTypePostgres)
	}
}

func TestPostgreSQLEngineGenerateReturnsError(t *testing.T) {
	engine := NewPostgreSQLEngine("postgres://localhost")

	_, err := engine.Generate(context.Background(), GenerateRequest{
		Role:        "readonly",
		Permissions: "SELECT",
	})
	if err == nil {
		t.Fatal("Generate = nil, want error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error = %q, want 'not implemented'", err.Error())
	}
}

func TestPostgreSQLEngineRevokeReturnsError(t *testing.T) {
	engine := NewPostgreSQLEngine("postgres://localhost")

	err := engine.Revoke(context.Background(), "lease-1")
	if err == nil {
		t.Fatal("Revoke = nil, want error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error = %q, want 'not implemented'", err.Error())
	}
}

func TestPostgreSQLEngineValidateReturnsError(t *testing.T) {
	engine := NewPostgreSQLEngine("postgres://localhost")

	err := engine.Validate(context.Background(), GenerateRequest{
		Role:        "readonly",
		Permissions: "SELECT",
	})
	if err == nil {
		t.Fatal("Validate = nil, want error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error = %q, want 'not implemented'", err.Error())
	}
}

func TestPostgreSQLEngineWithContext(t *testing.T) {
	engine := NewPostgreSQLEngine("postgres://localhost")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := engine.Generate(ctx, GenerateRequest{})
	if err == nil {
		t.Error("Generate with cancelled context = nil, want error")
	}
}
