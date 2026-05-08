package dynamicsecret

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

type mockSTSClient struct {
	assumeRoleFunc func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

func (m *mockSTSClient) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	if m.assumeRoleFunc != nil {
		return m.assumeRoleFunc(ctx, params, optFns...)
	}
	expiration := time.Now().Add(time.Hour)
	return &sts.AssumeRoleOutput{
		Credentials: &types.Credentials{
			AccessKeyId:     aws.String("AKIAIOSFODNN7EXAMPLE"),
			SecretAccessKey: aws.String("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
			SessionToken:    aws.String("session-token-value"),
			Expiration:      &expiration,
		},
	}, nil
}

func TestAWSSTSEngineImplementsInterface(t *testing.T) {
	var _ SecretEngine = NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")
}

func TestAWSSTSEngineType(t *testing.T) {
	engine := NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")
	if engine.Type() != EngineTypeAWSSTS {
		t.Errorf("Type() = %q, want %q", engine.Type(), EngineTypeAWSSTS)
	}
}

func TestAWSSTSEngineGenerate(t *testing.T) {
	var capturedInput *sts.AssumeRoleInput

	mock := &mockSTSClient{
		assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
			capturedInput = params
			expiration := time.Now().Add(time.Hour)
			return &sts.AssumeRoleOutput{
				Credentials: &types.Credentials{
					AccessKeyId:     aws.String("AKIAIOSFODNN7EXAMPLE"),
					SecretAccessKey: aws.String("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
					SessionToken:    aws.String("session-token-value"),
					Expiration:      &expiration,
				},
			}, nil
		},
	}

	engine := NewAWSSTSEngine("")
	engine.client = mock

	secret, err := engine.Generate(context.Background(), GenerateRequest{
		Role: "arn:aws:iam::123456789012:role/test",
		TTL:  time.Hour,
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if secret == nil {
		t.Fatal("secret is nil")
	}

	if capturedInput == nil {
		t.Fatal("AssumeRole was not called")
	}
	if capturedInput.RoleArn == nil || *capturedInput.RoleArn != "arn:aws:iam::123456789012:role/test" {
		t.Errorf("RoleArn = %v, want arn:aws:iam::123456789012:role/test", capturedInput.RoleArn)
	}
	if capturedInput.DurationSeconds == nil || *capturedInput.DurationSeconds != 3600 {
		t.Errorf("DurationSeconds = %v, want 3600", capturedInput.DurationSeconds)
	}
	if capturedInput.RoleSessionName == nil || *capturedInput.RoleSessionName == "" {
		t.Error("RoleSessionName is empty")
	}

	if secret.LeaseID == "" {
		t.Error("LeaseID is empty")
	}
	if secret.LeaseDuration != time.Hour {
		t.Errorf("LeaseDuration = %v, want 1h", secret.LeaseDuration)
	}
	if secret.Renewable {
		t.Error("Renewable should be false")
	}
	if secret.EngineType != EngineTypeAWSSTS {
		t.Errorf("EngineType = %q, want %q", secret.EngineType, EngineTypeAWSSTS)
	}

	if secret.Data["access_key_id"] != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("access_key_id = %v", secret.Data["access_key_id"])
	}
	if secret.Data["secret_access_key"] != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("secret_access_key = %v", secret.Data["secret_access_key"])
	}
	if secret.Data["session_token"] != "session-token-value" {
		t.Errorf("session_token = %v", secret.Data["session_token"])
	}
	if secret.Data["role_arn"] != "arn:aws:iam::123456789012:role/test" {
		t.Errorf("role_arn = %v", secret.Data["role_arn"])
	}
	if secret.Data["session_name"] == nil || secret.Data["session_name"] == "" {
		t.Error("session_name is empty")
	}
	if secret.Data["expiration"] == nil || secret.Data["expiration"] == "" {
		t.Error("expiration is empty")
	}
}

func TestAWSSTSEngineGenerateEmptyCredentials(t *testing.T) {
	mock := &mockSTSClient{
		assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
			return &sts.AssumeRoleOutput{
				Credentials: nil,
			}, nil
		},
	}

	engine := NewAWSSTSEngine("")
	engine.client = mock

	_, err := engine.Generate(context.Background(), GenerateRequest{
		Role: "arn:aws:iam::123456789012:role/test",
		TTL:  time.Hour,
	})
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
	if !strings.Contains(err.Error(), "empty credentials") {
		t.Errorf("error = %q, want 'empty credentials'", err.Error())
	}
}

func TestAWSSTSEngineGenerateDurationClamping(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		wantSecs int32
	}{
		{"below minimum", 5 * time.Minute, 900},
		{"exact minimum", 15 * time.Minute, 900},
		{"within range", time.Hour, 3600},
		{"above maximum", 48 * time.Hour, 43200},
		{"exact maximum", 12 * time.Hour, 43200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedDuration int32

			mock := &mockSTSClient{
				assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
					capturedDuration = *params.DurationSeconds
					expiration := time.Now().Add(time.Hour)
					return &sts.AssumeRoleOutput{
						Credentials: &types.Credentials{
							AccessKeyId:     aws.String("AKIAIOSFODNN7EXAMPLE"),
							SecretAccessKey: aws.String("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
							SessionToken:    aws.String("token"),
							Expiration:      &expiration,
						},
					}, nil
				},
			}

			engine := NewAWSSTSEngine("")
			engine.client = mock

			_, err := engine.Generate(context.Background(), GenerateRequest{
				Role: "arn:aws:iam::123456789012:role/test",
				TTL:  tt.ttl,
			})
			if err != nil {
				t.Fatalf("Generate error: %v", err)
			}

			if capturedDuration != tt.wantSecs {
				t.Errorf("DurationSeconds = %d, want %d", capturedDuration, tt.wantSecs)
			}
		})
	}
}

func TestAWSSTSEngineGenerateError(t *testing.T) {
	mock := &mockSTSClient{
		assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
			return nil, fmt.Errorf("api error: AccessDenied")
		},
	}

	engine := NewAWSSTSEngine("")
	engine.client = mock

	_, err := engine.Generate(context.Background(), GenerateRequest{
		Role: "arn:aws:iam::123456789012:role/test",
		TTL:  time.Hour,
	})
	if err == nil {
		t.Fatal("expected error from AssumeRole")
	}
	if !strings.Contains(err.Error(), "assume role") {
		t.Errorf("error = %q, want 'assume role'", err.Error())
	}
}

func TestAWSSTSEngineRevoke(t *testing.T) {
	engine := NewAWSSTSEngine("arn:aws:iam::123456789012:role/test")

	err := engine.Revoke(context.Background(), "any-lease-id")
	if err != nil {
		t.Errorf("Revoke should be no-op, got error: %v", err)
	}
}

func TestAWSSTSEngineValidate(t *testing.T) {
	engine := NewAWSSTSEngine("")

	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{"valid ARN", "arn:aws:iam::123456789012:role/test", false},
		{"valid govcloud ARN", "arn:aws-us-gov:iam::123456789012:role/test", false},
		{"empty role", "", true},
		{"no arn prefix", "arn:aws:ec2::123456789012:instance/i-abc", true},
		{"no role prefix", "arn:aws:iam::123456789012:user/me", true},
		{"garbage", "not-an-arn", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(context.Background(), GenerateRequest{
				Role: tt.role,
			})
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAWSSTSEngineValidateDefaultRoleARN(t *testing.T) {
	engine := NewAWSSTSEngine("arn:aws:iam::123456789012:role/default")

	err := engine.Validate(context.Background(), GenerateRequest{
		Role: "",
	})
	if err != nil {
		t.Errorf("expected no error with default role ARN, got: %v", err)
	}
}

func TestAWSSTSEngineContextCancellation(t *testing.T) {
	assumeCalled := false
	mock := &mockSTSClient{
		assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
			assumeCalled = true
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				expiration := time.Now().Add(time.Hour)
				return &sts.AssumeRoleOutput{
					Credentials: &types.Credentials{
						AccessKeyId:     aws.String("AKIAIOSFODNN7EXAMPLE"),
						SecretAccessKey: aws.String("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
						SessionToken:    aws.String("token"),
						Expiration:      &expiration,
					},
				}, nil
			}
		},
	}

	engine := NewAWSSTSEngine("")
	engine.client = mock

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := engine.Generate(ctx, GenerateRequest{
		Role: "arn:aws:iam::123456789012:role/test",
		TTL:  time.Hour,
	})
	if err == nil {
		t.Error("expected error with canceled context")
	}
	if assumeCalled && !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}
