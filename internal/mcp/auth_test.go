package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func pollWithTimeout(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestBearerAuthRejects401WithoutToken(t *testing.T) {
	handler := BearerAuthMiddleware("secret-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestBearerAuthRejectsWhenConfiguredTokenEmpty(t *testing.T) {
	handler := BearerAuthMiddleware("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestBearerAuthRejects401WithWrongToken(t *testing.T) {
	handler := BearerAuthMiddleware("secret-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestBearerAuthAcceptsValidToken(t *testing.T) {
	handler := BearerAuthMiddleware("secret-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestAgentHeaderRejects403WhenMissing(t *testing.T) {
	handler := AgentHeaderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestAgentHeaderSetsContext(t *testing.T) {
	var gotAgent string
	handler := AgentHeaderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAgent = AgentFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("X-OpenPass-Agent", "claude-code")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotAgent != "claude-code" {
		t.Fatalf("agent = %q, want %q", gotAgent, "claude-code")
	}
}

func TestRateLimiterStartCleanupPeriodic(t *testing.T) {
	rl := NewRateLimiter(5, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.2")

	stop := rl.StartCleanup(ctx, 30*time.Millisecond)
	defer stop()

	pollWithTimeout(t, func() bool {
		return rl.CleanupCount() >= 1
	}, 500*time.Millisecond, "expected at least 1 cleanup cycle")
}

func TestRateLimiterStartCleanupStopsOnCancel(t *testing.T) {
	rl := NewRateLimiter(5, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())

	_ = rl.StartCleanup(ctx, 20*time.Millisecond)
	pollWithTimeout(t, func() bool {
		return rl.CleanupCount() >= 0
	}, 100*time.Millisecond, "wait for cleanup goroutine to start")
	cancel()

	countAfterCancel := rl.CleanupCount()
	pollWithTimeout(t, func() bool {
		return rl.CleanupCount() == countAfterCancel
	}, 200*time.Millisecond, "CleanupCount should stay same after cancel")
}

func TestRateLimiterStartCleanupStopFunc(t *testing.T) {
	rl := NewRateLimiter(5, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := rl.StartCleanup(ctx, 20*time.Millisecond)
	pollWithTimeout(t, func() bool {
		return rl.CleanupCount() >= 0
	}, 100*time.Millisecond, "wait for cleanup goroutine to start")
	stop()

	countAfterStop := rl.CleanupCount()
	pollWithTimeout(t, func() bool {
		return rl.CleanupCount() == countAfterStop
	}, 200*time.Millisecond, "CleanupCount should stay same after stop")
}

func TestRateLimiterAllowNewIP(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	if !rl.Allow("192.168.1.1") {
		t.Fatal("expected first attempt to be allowed")
	}
}

func TestRateLimiterAllowWithinLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	ip := "192.168.1.2"

	for i := 0; i < 3; i++ {
		if !rl.Allow(ip) {
			t.Fatalf("expected attempt %d to be allowed", i+1)
		}
	}
}

func TestRateLimiterDeniesOverLimit(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	ip := "192.168.1.3"

	if !rl.Allow(ip) {
		t.Fatal("expected first attempt to be allowed")
	}
	if !rl.Allow(ip) {
		t.Fatal("expected second attempt to be allowed")
	}
	if rl.Allow(ip) {
		t.Fatal("expected third attempt to be denied")
	}
}

func TestRateLimiterAllowAfterWindow(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)
	ip := "192.168.1.4"

	if !rl.Allow(ip) {
		t.Fatal("expected first attempt to be allowed")
	}
	if !rl.Allow(ip) {
		t.Fatal("expected second attempt to be allowed")
	}
	if rl.Allow(ip) {
		t.Fatal("expected third attempt to be denied")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.Allow(ip) {
		t.Fatal("expected attempt after window to be allowed")
	}
}

func TestRateLimiterIndependentIPs(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	if !rl.Allow("10.0.0.1") {
		t.Fatal("expected 10.0.0.1 first attempt to be allowed")
	}
	if !rl.Allow("10.0.0.1") {
		t.Fatal("expected 10.0.0.1 second attempt to be allowed")
	}
	if rl.Allow("10.0.0.1") {
		t.Fatal("expected 10.0.0.1 third attempt to be denied")
	}

	if !rl.Allow("10.0.0.2") {
		t.Fatal("expected 10.0.0.2 first attempt to be allowed")
	}
	if !rl.Allow("10.0.0.2") {
		t.Fatal("expected 10.0.0.2 second attempt to be allowed")
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	rl := NewRateLimiter(5, 50*time.Millisecond)

	rl.Allow("192.168.1.10")
	rl.Allow("192.168.1.11")

	time.Sleep(60 * time.Millisecond)

	rl.Cleanup()

	if !rl.Allow("192.168.1.10") {
		t.Fatal("expected attempt after cleanup to be allowed for 192.168.1.10")
	}
	if !rl.Allow("192.168.1.11") {
		t.Fatal("expected attempt after cleanup to be allowed for 192.168.1.11")
	}
}

func TestRateLimiterCleanupOnlyExpired(t *testing.T) {
	rl := NewRateLimiter(5, time.Hour)

	rl.Allow("192.168.1.12")
	rl.Allow("192.168.1.13")

	rl.Cleanup()

	if !rl.Allow("192.168.1.12") {
		t.Fatal("expected second attempt to be allowed for 192.168.1.12")
	}
	if !rl.Allow("192.168.1.13") {
		t.Fatal("expected second attempt to be allowed for 192.168.1.13")
	}
}

func TestRateLimiterCleanupCount(t *testing.T) {
	rl := NewRateLimiter(5, 50*time.Millisecond)

	if rl.CleanupCount() != 0 {
		t.Fatalf("expected initial cleanup count to be 0, got %d", rl.CleanupCount())
	}

	rl.Allow("192.168.1.14")
	rl.Allow("192.168.1.15")
	rl.Allow("192.168.1.16")

	time.Sleep(60 * time.Millisecond)

	rl.cleanupOnce()

	if rl.CleanupCount() != 3 {
		t.Fatalf("expected cleanup count to be 3, got %d", rl.CleanupCount())
	}
}

func TestRateLimiterClose(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	if err := rl.Close(); err != nil {
		t.Fatalf("expected Close to return nil, got %v", err)
	}
}

func TestRateLimiterMiddlewareAllowsRequest(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)
	handler := RateLimiterMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestRateLimiterMiddlewareRateLimits(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	handler := RateLimiterMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest("POST", "/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
}

func TestRateLimiterMiddlewareDifferentIPs(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	handler := RateLimiterMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("10.0.0.1 request %d: status = %d, want 200", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("10.0.0.1 status = %d, want 429", rec.Code)
	}

	req2 := httptest.NewRequest("POST", "/mcp", nil)
	req2.RemoteAddr = "10.0.0.2:12345"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("10.0.0.2 status = %d, want 200", rec2.Code)
	}
}

func TestClientIPXForwardedFor(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := clientIP(req)
	if ip != "203.0.113.195" {
		t.Fatalf("clientIP = %q, want 203.0.113.195", ip)
	}
}

func TestClientIPXRealIP(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("X-Real-IP", "192.168.1.100")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := clientIP(req)
	if ip != "192.168.1.100" {
		t.Fatalf("clientIP = %q, want 192.168.1.100", ip)
	}
}

func TestClientIPRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.RemoteAddr = "192.168.1.50:54321"

	ip := clientIP(req)
	if ip != "192.168.1.50" {
		t.Fatalf("clientIP = %q, want 192.168.1.50", ip)
	}
}

func TestClientIPRemoteAddrNoPort(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.RemoteAddr = "192.168.1.50"

	ip := clientIP(req)
	if ip != "192.168.1.50" {
		t.Fatalf("clientIP = %q, want 192.168.1.50", ip)
	}
}

func TestClientIPUnknown(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.RemoteAddr = ""

	ip := clientIP(req)
	if ip != "unknown" {
		t.Fatalf("clientIP = %q, want unknown", ip)
	}
}

func TestRateLimiterConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100, time.Minute)
	ip := "192.168.1.100"

	var wg sync.WaitGroup
	allowed := make(chan bool, 200)

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- rl.Allow(ip)
		}()
	}

	wg.Wait()
	close(allowed)

	allowedCount := 0
	for a := range allowed {
		if a {
			allowedCount++
		}
	}

	if allowedCount != 100 {
		t.Fatalf("expected 100 allowed requests, got %d", allowedCount)
	}
}
