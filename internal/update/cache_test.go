package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestCachePathTraversalPrevention(t *testing.T) {
	traversalPaths := []string{
		"../update-cache.json",
		"/tmp/../../../etc/passwd",
		"foo/../../bar/cache.json",
	}

	for _, path := range traversalPaths {
		cache := NewCacheWithTTL(path, 24*time.Hour)

		err := cache.Save(&CacheEntry{
			Timestamp:     time.Now(),
			LatestVersion: "1.0.0",
			ReleaseURL:    "https://example.com",
		})
		if err == nil {
			t.Fatalf("Save() with path %q should have failed", path)
		}
		if !strings.Contains(err.Error(), "invalid cache path") {
			t.Fatalf("Save() with path %q returned unexpected error: %v", path, err)
		}

		_, err = cache.Load()
		if err == nil {
			t.Fatalf("Load() with path %q should have failed", path)
		}
		if !strings.Contains(err.Error(), "invalid cache path") {
			t.Fatalf("Load() with path %q returned unexpected error: %v", path, err)
		}
	}
}

func TestCacheDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "subdir", "nested", "update-cache.json")

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	entry := &CacheEntry{
		Timestamp:     time.Now(),
		LatestVersion: "1.2.0",
		ReleaseURL:    "https://example.com/v1.2.0",
	}

	if err := cache.Save(entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache file should exist after Save(): %v", err)
	}

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("Load() returned nil, want entry")
	}
}

func TestCacheEmptyPath(t *testing.T) {
	cache := &Cache{path: "", ttl: 24 * time.Hour}

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != nil {
		t.Fatal("Load() with empty path should return nil")
	}

	err = cache.Save(&CacheEntry{
		Timestamp:     time.Now(),
		LatestVersion: "1.0.0",
		ReleaseURL:    "https://example.com",
	})
	if err != nil {
		t.Fatalf("Save() with empty path should not error, got: %v", err)
	}
}

func TestCachePathAndTTLAccessors(t *testing.T) {
	expectedPath := "/tmp/test-cache.json"
	expectedTTL := 48 * time.Hour

	cache := NewCacheWithTTL(expectedPath, expectedTTL)

	if cache.Path() != expectedPath {
		t.Fatalf("Path() = %q, want %q", cache.Path(), expectedPath)
	}
	if cache.TTL() != expectedTTL {
		t.Fatalf("TTL() = %v, want %v", cache.TTL(), expectedTTL)
	}
}

func TestCacheZeroOrNegativeTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	cache := NewCacheWithTTL(cachePath, 0)
	if cache.TTL() != DefaultCacheTTL {
		t.Fatalf("TTL with 0 = %v, want %v", cache.TTL(), DefaultCacheTTL)
	}

	cache = NewCacheWithTTL(cachePath, -1*time.Hour)
	if cache.TTL() != DefaultCacheTTL {
		t.Fatalf("TTL with negative = %v, want %v", cache.TTL(), DefaultCacheTTL)
	}
}

func TestCacheLoadReadError(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	if err := os.Mkdir(cachePath, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	loaded, err := cache.Load()
	if err == nil {
		t.Fatal("Load() with directory as path should error")
	}
	if loaded != nil {
		t.Fatal("Load() should return nil on error")
	}
}

func TestCacheSaveNilEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	if err := cache.Save(nil); err != nil {
		t.Fatalf("Save(nil) error = %v", err)
	}

	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatal("Save(nil) should not create a file")
	}
}

func TestCacheSaveWriteError(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "readonly", "update-cache.json")

	if err := os.MkdirAll(filepath.Dir(cachePath), 0o500); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	defer func() {
		_ = os.Chmod(filepath.Dir(cachePath), 0o700)
	}()

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	err := cache.Save(&CacheEntry{
		Timestamp:     time.Now(),
		LatestVersion: "1.0.0",
		ReleaseURL:    "https://example.com",
	})
	if err == nil {
		t.Fatal("Save() with read-only dir should error")
	}
}

func TestCacheInvalidateNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	if err := cache.Invalidate(); err != nil {
		t.Fatalf("Invalidate() on non-existent file error = %v", err)
	}
}

func TestCacheInvalidateError(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	if err := os.WriteFile(cachePath, []byte("test"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := os.Chmod(tmpDir, 0o500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	defer func() {
		_ = os.Chmod(tmpDir, 0o700)
	}()

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	err := cache.Invalidate()
	if err == nil {
		t.Fatal("Invalidate() with read-only dir should error")
	}
}

func TestCacheZeroLengthFile(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "update-cache.json")

	if err := os.WriteFile(cachePath, []byte{}, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cache := NewCacheWithTTL(cachePath, 24*time.Hour)

	loaded, err := cache.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != nil {
		t.Fatal("Load() with empty file should return nil")
	}
}

func TestParseStableVersionZeroVersion(t *testing.T) {
	v, ok := parseStableVersion("0.0.0")
	if !ok {
		t.Fatal("parseStableVersion(\"0.0.0\") should return ok=true")
	}
	if v.major != 0 || v.minor != 0 || v.patch != 0 {
		t.Fatalf("version = {%d,%d,%d}, want {0,0,0}", v.major, v.minor, v.patch)
	}
}

func TestCompareStableVersionsEqual(t *testing.T) {
	left, ok := parseStableVersion("1.2.3")
	if !ok {
		t.Fatal("parseStableVersion(\"1.2.3\") failed")
	}
	right, ok := parseStableVersion("1.2.3")
	if !ok {
		t.Fatal("parseStableVersion(\"1.2.3\") failed")
	}

	result := compareStableVersions(left, right)
	if result != 0 {
		t.Fatalf("compareStableVersions({1,2,3}, {1,2,3}) = %d, want 0", result)
	}
}

func TestStableVersionString(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"1.2.3", "1.2.3"},
		{"0.0.0", "0.0.0"},
		{"v10.20.30", "10.20.30"},
	}

	for _, tt := range tests {
		v, ok := parseStableVersion(tt.raw)
		if !ok {
			t.Fatalf("parseStableVersion(%q) failed", tt.raw)
		}
		if v.String() != tt.want {
			t.Fatalf("parseStableVersion(%q).String() = %q, want %q", tt.raw, v.String(), tt.want)
		}
	}
}
