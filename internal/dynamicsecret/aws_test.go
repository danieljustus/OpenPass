package dynamicsecret

import (
	"context"
	"strings"
	"testing"
)

func TestAWSSTSEngineImplementsInterface(t *testing.T) {
	var _ SecretEngine = NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")
}

func TestAWSSTSEngineType(t *testing.T) {
	engine := NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")
	if engine.Type() != EngineTypeAWSSTS {
		t.Errorf("Type() = %q, want %q", engine.Type(), EngineTypeAWSSTS)
	}
}

func TestAWSSTSEngineGenerateReturnsError(t *testing.T) {
	engine := NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")

	_, err := engine.Generate(context.Background(), GenerateRequest{
		Role:        "arn:aws:iam::123456789012:role/test",
		Permissions: "sts:AssumeRole",
	})
	if err == nil {
		t.Fatal("Generate = nil, want error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error = %q, want 'not implemented'", err.Error())
	}
}

func TestAWSSTSEngineRevokeReturnsError(t *testing.T) {
	engine := NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")

	err := engine.Revoke(context.Background(), "lease-1")
	if err == nil {
		t.Fatal("Revoke = nil, want error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error = %q, want 'not implemented'", err.Error())
	}
}

func TestAWSSTSEngineValidateReturnsError(t *testing.T) {
	engine := NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")

	err := engine.Validate(context.Background(), GenerateRequest{
		Role:        "arn:aws:iam::123456789012:role/test",
		Permissions: "sts:AssumeRole",
	})
	if err == nil {
		t.Fatal("Validate = nil, want error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error = %q, want 'not implemented'", err.Error())
	}
}

func TestAWSSTSEngineWithContext(t *testing.T) {
	engine := NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := engine.Generate(ctx, GenerateRequest{})
	if err == nil {
		t.Error("Generate with cancelled context = nil, want error")
	}
}
