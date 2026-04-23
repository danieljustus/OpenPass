package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	entry := &CacheEntry{
		Timestamp:     time.Now(),
		LatestVersion: "1.2.0",
		ReleaseURL:    "https://example.com/v1.2.0",
	}

	if err := cache.Save(entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("Load() returned nil, want entry")
	}
	if loaded.LatestVersion != entry.LatestVersion {
		t.Fatalf("LatestVersion = %q, want %q", loaded.LatestVersion, entry.LatestVersion)
	}
	if loaded.ReleaseURL != entry.ReleaseURL {
		t.Fatalf("ReleaseURL = %q, want %q", loaded.ReleaseURL, entry.ReleaseURL)
	}
}

func TestCacheExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	cache := NewCacheWithTTL(cachePath, 1*time.Hour)

	entry := &CacheEntry{
		Timestamp:     time.Now().Add(-2 * time.Hour),
		LatestVersion: "1.2.0",
		ReleaseURL:    "https://example.com/v1.2.0",
	}

	if err := cache.Save(entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != nil {
		t.Fatalf("Load() returned entry for expired cache, want nil")
	}
}

func TestCacheNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent.json")

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != nil {
		t.Fatalf("Load() returned entry for non-existent cache, want nil")
	}
}

func TestCacheCorrupted(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	if err := os.WriteFile(cachePath, []byte("invalid json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != nil {
		t.Fatalf("Load() returned entry for corrupted cache, want nil")
	}
}

func TestCacheInvalidate(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	entry := &CacheEntry{
		Timestamp:     time.Now(),
		LatestVersion: "1.2.0",
		ReleaseURL:    "https://example.com/v1.2.0",
	}

	if err := cache.Save(entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := cache.Invalidate(); err != nil {
		t.Fatalf("Invalidate() error = %v", err)
	}

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != nil {
		t.Fatalf("Load() returned entry after invalidation, want nil")
	}
}

func TestCacheDefaultTTL(t *testing.T) {
	cache := NewCache()
	if cache.TTL() != DefaultCacheTTL {
		t.Fatalf("TTL = %v, want %v", cache.TTL(), DefaultCacheTTL)
	}
}

func TestCacheCustomTTL(t *testing.T) {
	customTTL := 12 * time.Hour
	cache := NewCacheWithTTL("", customTTL)
	if cache.TTL() != customTTL {
		t.Fatalf("TTL = %v, want %v", cache.TTL(), customTTL)
	}
}

func TestCacheZeroTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	data, _ := json.Marshal(&CacheEntry{
		Timestamp:     time.Time{},
		LatestVersion: "1.2.0",
		ReleaseURL:    "https://example.com/v1.2.0",
	})
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != nil {
		t.Fatalf("Load() returned entry with zero timestamp, want nil")
	}
}
