package dynamicsecret

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

type mockResult struct {
	rowsAffected int64
}

func (r *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (r *mockResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type mockDB struct {
	execContextFunc func(ctx context.Context, query string, args ...any) (sql.Result, error)
	closeFunc       func() error
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if m.execContextFunc != nil {
		return m.execContextFunc(ctx, query, args...)
	}
	return &mockResult{}, nil
}

func (m *mockDB) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestPostgreSQLEngineImplementsInterface(t *testing.T) {
	var _ SecretEngine = NewPostgreSQLEngine("postgres://localhost")
}

func TestPostgreSQLEngineType(t *testing.T) {
	engine := NewPostgreSQLEngine("postgres://localhost")
	if engine.Type() != EngineTypePostgres {
		t.Errorf("Type() = %q, want %q", engine.Type(), EngineTypePostgres)
	}
}

func TestPostgreSQLEngineGenerate(t *testing.T) {
	var executedQueries []string
	var capturedArgs [][]any

	db := &mockDB{
		execContextFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			executedQueries = append(executedQueries, query)
			capturedArgs = append(capturedArgs, args)
			return &mockResult{}, nil
		},
	}

	engine := NewPostgreSQLEngine("postgres://admin:pass@localhost:5432/testdb")
	engine.db = db

	secret, err := engine.Generate(context.Background(), GenerateRequest{
		Role: "readonly",
		TTL:  30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if secret == nil {
		t.Fatal("secret is nil")
	}

	if len(executedQueries) != 2 {
		t.Fatalf("expected 2 queries, got %d: %v", len(executedQueries), executedQueries)
	}

	createQ := executedQueries[0]
	if !strings.HasPrefix(createQ, "CREATE USER ") {
		t.Errorf("first query = %q, want CREATE USER ...", createQ)
	}
	if !strings.Contains(createQ, "WITH PASSWORD $1") {
		t.Errorf("first query missing PASSWORD $1: %q", createQ)
	}
	if !strings.Contains(createQ, "VALID UNTIL $2") {
		t.Errorf("first query missing VALID UNTIL $2: %q", createQ)
	}

	grantQ := executedQueries[1]
	if !strings.HasPrefix(grantQ, "GRANT ") {
		t.Errorf("second query = %q, want GRANT ...", grantQ)
	}
	if !strings.Contains(grantQ, `"readonly"`) {
		t.Errorf("grant query missing role: %q", grantQ)
	}

	if secret.LeaseID == "" {
		t.Error("LeaseID is empty")
	}
	if secret.LeaseDuration != 30*time.Minute {
		t.Errorf("LeaseDuration = %v, want 30m", secret.LeaseDuration)
	}
	if secret.Renewable {
		t.Error("Renewable should be false")
	}
	if secret.EngineType != EngineTypePostgres {
		t.Errorf("EngineType = %q, want %q", secret.EngineType, EngineTypePostgres)
	}
	if secret.Data["username"] == nil {
		t.Error("username not in data")
	}
	if secret.Data["password"] == nil {
		t.Error("password not in data")
	}
	if secret.Data["password"] == "" {
		t.Error("password is empty")
	}
	if secret.Data["role"] != "readonly" {
		t.Errorf("role = %v, want readonly", secret.Data["role"])
	}
	if secret.Data["connection_string"] != "postgres://admin:pass@localhost:5432/testdb" {
		t.Errorf("connection_string = %v", secret.Data["connection_string"])
	}
	if got, ok := secret.Data["password"].(string); !ok || len(got) != 24 {
		t.Errorf("password length = %d, want 24", len(got))
	}

	if len(capturedArgs) >= 1 {
		args := capturedArgs[0]
		if len(args) != 2 {
			t.Errorf("CREATE USER args count = %d, want 2 (password, valid_until)", len(args))
		}
	}
}

func TestPostgreSQLEngineGenerateGrantFailure(t *testing.T) {
	callCount := 0
	db := &mockDB{
		execContextFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("grant failed")
			}
			return &mockResult{}, nil
		},
	}

	engine := NewPostgreSQLEngine("postgres://localhost")
	engine.db = db

	_, err := engine.Generate(context.Background(), GenerateRequest{
		Role: "readonly",
		TTL:  time.Hour,
	})
	if err == nil {
		t.Fatal("expected error on grant failure")
	}
	if !strings.Contains(err.Error(), "grant role") {
		t.Errorf("error = %q, want 'grant role'", err.Error())
	}

	if callCount != 3 {
		t.Errorf("expected 3 queries (create, grant, drop), got %d", callCount)
	}
}

func TestPostgreSQLEngineRevoke(t *testing.T) {
	var executedQueries []string

	db := &mockDB{
		execContextFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			executedQueries = append(executedQueries, query)
			return &mockResult{}, nil
		},
	}

	engine := NewPostgreSQLEngine("postgres://localhost")
	engine.db = db

	secret, err := engine.Generate(context.Background(), GenerateRequest{
		Role: "readonly",
		TTL:  time.Hour,
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	executedQueries = nil

	err = engine.Revoke(context.Background(), secret.LeaseID)
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	if len(executedQueries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(executedQueries))
	}

	dropQ := executedQueries[0]
	if !strings.HasPrefix(dropQ, "DROP USER IF EXISTS ") {
		t.Errorf("query = %q, want DROP USER IF EXISTS ...", dropQ)
	}

	err = engine.Revoke(context.Background(), secret.LeaseID)
	if err == nil {
		t.Error("expected error on second revoke")
	}
}

func TestPostgreSQLEngineRevokeLeaseNotFound(t *testing.T) {
	db := &mockDB{}
	engine := NewPostgreSQLEngine("postgres://localhost")
	engine.db = db

	err := engine.Revoke(context.Background(), "nonexistent-lease")
	if err == nil {
		t.Fatal("expected error for nonexistent lease")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err.Error())
	}
}

func TestPostgreSQLEngineValidate(t *testing.T) {
	engine := NewPostgreSQLEngine("postgres://localhost")

	tests := []struct {
		name    string
		role    string
		ttl     time.Duration
		wantErr bool
	}{
		{"valid", "readonly", time.Hour, false},
		{"empty role", "", time.Hour, true},
		{"zero TTL", "readonly", 0, true},
		{"negative TTL", "readonly", -time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(context.Background(), GenerateRequest{
				Role: tt.role,
				TTL:  tt.ttl,
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

func TestPostgreSQLEngineContextCancellation(t *testing.T) {
	execCalled := false
	db := &mockDB{
		execContextFunc: func(ctx context.Context, query string, args ...any) (sql.Result, error) {
			execCalled = true
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &mockResult{}, nil
			}
		},
	}

	engine := NewPostgreSQLEngine("postgres://localhost")
	engine.db = db

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := engine.Generate(ctx, GenerateRequest{
		Role: "readonly",
		TTL:  time.Hour,
	})
	if err == nil {
		t.Error("expected error with canceled context")
	}
	if execCalled && !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestPostgreSQLEngineUsernameFormat(t *testing.T) {
	username := generateUsername("readonly")
	if !strings.HasPrefix(username, "readonly_") {
		t.Errorf("username = %q, want prefix readonly_", username)
	}
	if len(username) <= len("readonly_") {
		t.Errorf("username too short: %q", username)
	}
}

func TestPostgreSQLEnginePasswordGeneration(t *testing.T) {
	password, err := generatePassword(24)
	if err != nil {
		t.Fatalf("generatePassword error: %v", err)
	}
	if len(password) != 24 {
		t.Errorf("password length = %d, want 24", len(password))
	}

	for _, c := range password {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", c) {
			t.Errorf("password contains invalid character: %c", c)
		}
	}
}

func TestPostgreSQLEngineQuoteIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", `"simple"`},
		{"with\"quote", `"with""quote"`},
		{"", `""`},
	}
	for _, tt := range tests {
		got := quoteIdentifier(tt.input)
		if got != tt.want {
			t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
