//go:build !dynamic_secrets

package dynamicsecret

import (
	"context"
	"errors"
)

// AWSSTSEngine is a stub that returns an error when dynamic secrets are not compiled in.
type AWSSTSEngine struct{}

// NewAWSSTSEngine creates a stub AWS STS engine.
func NewAWSSTSEngine(_ string) *AWSSTSEngine {
	return &AWSSTSEngine{}
}

// Type returns the engine type identifier.
func (e *AWSSTSEngine) Type() string {
	return EngineTypeAWSSTS
}

// Generate returns an error indicating dynamic secrets support is not compiled in.
func (e *AWSSTSEngine) Generate(_ context.Context, _ GenerateRequest) (*Secret, error) {
	return nil, errors.New("dynamic secrets not compiled in (build with -tags dynamic_secrets)")
}

// Revoke returns an error indicating dynamic secrets support is not compiled in.
func (e *AWSSTSEngine) Revoke(_ context.Context, _ string) error {
	return errors.New("dynamic secrets not compiled in (build with -tags dynamic_secrets)")
}

// Validate returns an error indicating dynamic secrets support is not compiled in.
func (e *AWSSTSEngine) Validate(_ context.Context, _ GenerateRequest) error {
	return errors.New("dynamic secrets not compiled in (build with -tags dynamic_secrets)")
}
