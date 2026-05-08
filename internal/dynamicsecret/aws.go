package dynamicsecret

import (
	"context"
	"errors"
)

// AWSSTSEngine is a stub implementation of SecretEngine for AWS STS credentials.
// Wave 1: interface methods compile but do not call the real AWS API.
type AWSSTSEngine struct {
	roleARN string
}

// NewAWSSTSEngine creates a new AWS STS engine stub.
func NewAWSSTSEngine(roleARN string) *AWSSTSEngine {
	return &AWSSTSEngine{roleARN: roleARN}
}

// Type returns the engine type identifier.
func (e *AWSSTSEngine) Type() string {
	return EngineTypeAWSSTS
}

// Generate returns an error indicating the engine is not yet implemented.
func (e *AWSSTSEngine) Generate(ctx context.Context, req GenerateRequest) (*Secret, error) {
	return nil, errors.New("aws-sts engine not implemented")
}

// Revoke returns an error indicating the engine is not yet implemented.
func (e *AWSSTSEngine) Revoke(ctx context.Context, leaseID string) error {
	return errors.New("aws-sts engine not implemented")
}

// Validate returns an error indicating the engine is not yet implemented.
func (e *AWSSTSEngine) Validate(ctx context.Context, req GenerateRequest) error {
	return errors.New("aws-sts engine not implemented")
}
