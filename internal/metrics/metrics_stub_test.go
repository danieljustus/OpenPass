//go:build !metrics

package metrics

import (
	"context"
	"testing"
	"time"
)

func TestStubRecordMCPRequestNoPanic(t *testing.T) {
	RecordMCPRequest("test", "agent", "success", time.Second)
}

func TestStubRecordAuthDenialNoPanic(t *testing.T) {
	RecordAuthDenial("reason", "agent")
}

func TestStubRecordApprovalNoPanic(t *testing.T) {
	RecordApproval("agent", "granted")
}

func TestStubRecordVaultOperationNoPanic(t *testing.T) {
	RecordVaultOperation("read", "success")
}

func TestStubRegistryReturnsNonNil(t *testing.T) {
	reg := Registry()
	if reg == nil {
		t.Fatal("Registry() returned nil")
	}
}

func TestStubInitTracingReturnsNoop(t *testing.T) {
	shutdown, err := InitTracing("", "test")
	if err != nil {
		t.Fatalf("InitTracing error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown is nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestStubTracerReturnsNonNil(t *testing.T) {
	tr := Tracer()
	if tr == nil {
		t.Error("Tracer() returned nil")
	}
}

func TestStubHashEntryPath(t *testing.T) {
	h1 := HashEntryPath("github/token")
	h2 := HashEntryPath("github/token")
	h3 := HashEntryPath("other/path")
	if h1 != h2 {
		t.Error("expected same hash for same input")
	}
	if h1 == h3 {
		t.Error("expected different hash for different input")
	}
	if h1 == "" {
		t.Error("expected non-empty hash")
	}
}

func TestStubStartSpan(t *testing.T) {
	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-span")
	if newCtx == nil {
		t.Error("expected non-nil context")
	}
	if span == nil {
		t.Error("expected non-nil span")
	}
	span.End()
}

func TestStubSpanFromContext(t *testing.T) {
	span := SpanFromContext(context.Background())
	if span == nil {
		t.Error("expected non-nil span")
	}
}

func TestStubRecordAllNoPanic(t *testing.T) {
	RecordVaultEntryCount("/tmp/test", 5)
	RecordVaultOperationDuration("open", time.Millisecond)
	RecordSessionCacheEvent("hit")
	RecordIdentityCacheEvent("miss")
	RecordUpdateCheck("up_to_date")
	RecordPolicyEvalDuration(time.Millisecond)
}
