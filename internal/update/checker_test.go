package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type stubHTTPDoer struct {
	do func(req *http.Request) (*http.Response, error)
}

func (s stubHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

func TestCheckerSkipsNonReleaseVersions(t *testing.T) {
	checker := NewChecker(stubHTTPDoer{
		do: func(req *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected HTTP request to %s", req.URL.String())
			return nil, nil
		},
	})

	result, err := checker.Check(context.Background(), "dev")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.Checkable {
		t.Fatalf("Checkable = %v, want false", result.Checkable)
	}
	if result.CurrentVersion != "dev" {
		t.Fatalf("CurrentVersion = %q, want %q", result.CurrentVersion, "dev")
	}
}

func TestCheckerReportsAvailableUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.0","html_url":"https://example.com/v1.2.0"}`))
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL

	result, err := checker.Check(context.Background(), "1.0.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !result.Checkable {
		t.Fatal("expected release build to be checkable")
	}
	if !result.UpdateAvailable {
		t.Fatal("expected update to be available")
	}
	if result.LatestVersion != "1.2.0" {
		t.Fatalf("LatestVersion = %q, want %q", result.LatestVersion, "1.2.0")
	}
	if result.ReleaseURL != "https://example.com/v1.2.0" {
		t.Fatalf("ReleaseURL = %q", result.ReleaseURL)
	}
}

func TestCheckerReportsUpToDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.10.0","html_url":"https://example.com/v1.10.0"}`))
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL

	result, err := checker.Check(context.Background(), "v1.10.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.UpdateAvailable {
		t.Fatal("expected no update to be available")
	}
	if result.CurrentVersion != "1.10.0" {
		t.Fatalf("CurrentVersion = %q, want %q", result.CurrentVersion, "1.10.0")
	}
}

func TestCheckerRejectsInvalidLatestTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"latest","html_url":"https://example.com/latest"}`))
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL
	checker.Cache = nil

	_, err := checker.Check(context.Background(), "1.0.0")
	if err == nil {
		t.Fatal("expected invalid tag name to fail")
	}
	if !strings.Contains(err.Error(), "stable semantic version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckerReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL
	checker.Cache = nil

	_, err := checker.Check(context.Background(), "1.0.0")
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckerReturnsDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":`))
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL
	checker.Cache = nil

	_, err := checker.Check(context.Background(), "1.0.0")
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode latest release response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckerReturnsTimeoutError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.1","html_url":"https://example.com/v1.0.1"}`))
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 10 * time.Millisecond

	checker := NewChecker(client)
	checker.LatestReleaseURL = server.URL
	checker.Cache = nil

	_, err := checker.Check(context.Background(), "1.0.0")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "request latest release") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompareStableVersions(t *testing.T) {
	left, ok := parseStableVersion("1.10.0")
	if !ok {
		t.Fatal("expected left version to parse")
	}
	right, ok := parseStableVersion("1.2.0")
	if !ok {
		t.Fatal("expected right version to parse")
	}

	if compareStableVersions(left, right) <= 0 {
		t.Fatalf("expected %s to be newer than %s", left.String(), right.String())
	}
}

func TestCheckerUsesCache(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCacheWithTTL(tmpDir+"/update-cache.json", 24*time.Hour)

	_ = cache.Save(&CacheEntry{
		Timestamp:     time.Now(),
		LatestVersion: "1.5.0",
		ReleaseURL:    "https://example.com/v1.5.0",
	})

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.5.0","html_url":"https://example.com/v1.5.0"}`))
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL
	checker.Cache = cache

	result, err := checker.Check(context.Background(), "1.0.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if requestCount != 0 {
		t.Fatalf("expected cache hit, but got %d HTTP requests", requestCount)
	}
	if result.LatestVersion != "1.5.0" {
		t.Fatalf("LatestVersion = %q, want %q", result.LatestVersion, "1.5.0")
	}
	if result.ReleaseURL != "https://example.com/v1.5.0" {
		t.Fatalf("ReleaseURL = %q, want %q", result.ReleaseURL, "https://example.com/v1.5.0")
	}
}

func TestCheckerForceBypassesCache(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCacheWithTTL(tmpDir+"/update-cache.json", 24*time.Hour)

	_ = cache.Save(&CacheEntry{
		Timestamp:     time.Now(),
		LatestVersion: "1.5.0",
		ReleaseURL:    "https://example.com/v1.5.0",
	})

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.6.0","html_url":"https://example.com/v1.6.0"}`))
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL
	checker.Cache = cache

	result, err := checker.CheckWithForce(context.Background(), "1.0.0", true)
	if err != nil {
		t.Fatalf("CheckWithForce() error = %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected 1 HTTP request with --force, got %d", requestCount)
	}
	if result.LatestVersion != "1.6.0" {
		t.Fatalf("LatestVersion = %q, want %q", result.LatestVersion, "1.6.0")
	}
}

func TestCheckerSavesToCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := tmpDir + "/update-cache.json"
	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.0","html_url":"https://example.com/v1.2.0"}`))
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL
	checker.Cache = cache

	_, err := checker.Check(context.Background(), "1.0.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("expected cache to be saved")
	}
	if loaded.LatestVersion != "1.2.0" {
		t.Fatalf("cached LatestVersion = %q, want %q", loaded.LatestVersion, "1.2.0")
	}
	if loaded.ReleaseURL != "https://example.com/v1.2.0" {
		t.Fatalf("cached ReleaseURL = %q, want %q", loaded.ReleaseURL, "https://example.com/v1.2.0")
	}
}

func TestCheckerNilCache(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.0","html_url":"https://example.com/v1.2.0"}`))
	}))
	defer server.Close()

	checker := NewChecker(server.Client())
	checker.LatestReleaseURL = server.URL
	checker.Cache = nil

	result, err := checker.Check(context.Background(), "1.0.0")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected 1 HTTP request with nil cache, got %d", requestCount)
	}
	if result.LatestVersion != "1.2.0" {
		t.Fatalf("LatestVersion = %q, want %q", result.LatestVersion, "1.2.0")
	}
}
