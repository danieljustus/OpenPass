package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogEntryWritesJSONL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := New("test-agent")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.LogEntry(LogEntry{
		Agent:     "test-agent",
		Action:    "get",
		Path:      "dev/api-key",
		Transport: "stdio",
		OK:        true,
		DurMs:     42,
	})

	content, err := os.ReadFile(filepath.Join(home, ".openpass", "audit-test-agent.log"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	line := strings.TrimSpace(string(content))
	var entry map[string]any
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("invalid JSON: %v\nline: %s", err, line)
	}

	if entry["agent"] != "test-agent" {
		t.Fatalf("agent = %v, want test-agent", entry["agent"])
	}
	if entry["action"] != "get" {
		t.Fatalf("action = %v, want get", entry["action"])
	}
	if entry["ok"] != true {
		t.Fatalf("ok = %v, want true", entry["ok"])
	}
	if entry["transport"] != "stdio" {
		t.Fatalf("transport = %v, want stdio", entry["transport"])
	}
	if _, ok := entry["ts"]; !ok {
		t.Fatal("missing ts field")
	}
}

func TestLogEntryDenialIncludesReason(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := New("test-agent")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.LogEntry(LogEntry{
		Agent:     "test-agent",
		Action:    "set",
		Path:      "secret/key",
		Field:     "password",
		Transport: "http",
		OK:        false,
		Reason:    "write_denied",
	})

	content, err := os.ReadFile(filepath.Join(home, ".openpass", "audit-test-agent.log"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(content))), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if entry["ok"] != false {
		t.Fatalf("ok = %v, want false", entry["ok"])
	}
	if entry["reason"] != "write_denied" {
		t.Fatalf("reason = %v, want write_denied", entry["reason"])
	}
	if entry["field"] != "password" {
		t.Fatalf("field = %v, want password", entry["field"])
	}
}

func TestLoggerCloseNilSafety(t *testing.T) {
	var l *Logger
	if err := l.Close(); err != nil {
		t.Fatalf("Close() on nil logger should not error, got %v", err)
	}
}

func TestLogEntryNilLoggerSafety(t *testing.T) {
	var l *Logger
	// Should not panic
	l.LogEntry(LogEntry{
		Agent:  "test",
		Action: "get",
		OK:     true,
	})
}

func TestLogEntryNilFileSafety(t *testing.T) {
	l := &Logger{file: nil}
	// Should not panic
	l.LogEntry(LogEntry{
		Agent:  "test",
		Action: "get",
		OK:     true,
	})
}

func TestLogEntryMultipleWrites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := New("multi-write")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	for i := 0; i < 3; i++ {
		logger.LogEntry(LogEntry{
			Agent:  "test-agent",
			Action: "get",
			Path:   "test/path",
			OK:     i%2 == 0,
			DurMs:  int64(i * 10),
		})
	}

	content, err := os.ReadFile(filepath.Join(home, ".openpass", "audit-multi-write.log"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid JSON on line %d: %v", i, err)
		}
	}
}

func TestLogEntryAutoTimestamp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := New("timestamp-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Entry without timestamp - should get auto-filled
	logger.LogEntry(LogEntry{
		Agent:  "test-agent",
		Action: "list",
		OK:     true,
	})

	content, err := os.ReadFile(filepath.Join(home, ".openpass", "audit-timestamp-test.log"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(content))), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := entry["ts"]; !ok {
		t.Fatal("expected auto-generated ts field")
	}
}

func TestLogEntryPreservesProvidedTimestamp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := New("preserved-ts")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	customTS := "2024-01-15T10:30:00Z"
	logger.LogEntry(LogEntry{
		Timestamp: customTS,
		Agent:     "test-agent",
		Action:    "get",
		OK:        true,
	})

	content, err := os.ReadFile(filepath.Join(home, ".openpass", "audit-preserved-ts.log"))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(content))), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if entry["ts"] != customTS {
		t.Fatalf("expected ts=%s, got %v", customTS, entry["ts"])
	}
}

func TestNewRejectsPathSeparator(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := New("agent/with/slash")
	if err == nil {
		t.Fatal("expected error for agent name with slash")
	}
}

func TestNewRejectsBackslash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := New("agent\\with\\backslash")
	if err == nil {
		t.Fatal("expected error for agent name with backslash")
	}
}

func TestNewRejectsDotDot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := New("..")
	if err == nil {
		t.Fatal("expected error for .. agent name")
	}
}

func TestNewRejectsDot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := New(".")
	if err == nil {
		t.Fatal("expected error for . agent name")
	}
}

func TestNewRejectsDotDotInName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := New("my..agent")
	if err == nil {
		t.Fatal("expected error for agent name containing ..")
	}
}

func TestNewCreatesAuditDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := New("create-dir-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	expectedDir := filepath.Join(home, ".openpass")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Fatalf("audit directory was not created at %s", expectedDir)
	}
}

func TestNewCreatesLogFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := New("logfile-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	expectedFile := filepath.Join(home, ".openpass", "audit-logfile-test.log")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("audit log file was not created at %s", expectedFile)
	}
}

func TestNewSetsCorrectPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := New("perms-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	expectedFile := filepath.Join(home, ".openpass", "audit-perms-test.log")
	info, err := os.Stat(expectedFile)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Fatalf("expected file permissions 0o600, got %o", perm)
	}
}

func TestNewUsesUserHomeDirWhenHomeEmpty(t *testing.T) {
	oldHome := os.Getenv("HOME")
	//nolint:errcheck // best-effort cleanup in test
	os.Unsetenv("HOME")
	defer func() {
		if oldHome != "" {
			//nolint:errcheck // best-effort restore in test
			os.Setenv("HOME", oldHome)
		}
	}()

	// os.UserHomeDir may fail in some environments; skip if so
	logger, err := New("home-dir-test")
	if err != nil && strings.Contains(err.Error(), "not defined") {
		t.Skip("HOME not available in this environment")
	}
	if err == nil {
		defer func() { _ = logger.Close() }()
	}
}

func TestNewErrorWhenAuditDirNotCreatable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod 0 has no effect")
	}

	parent := t.TempDir()
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	defer os.Chmod(parent, 0o700) //nolint:errcheck

	t.Setenv("HOME", parent)

	_, err := New("dir-fail-agent")
	if err == nil {
		t.Fatal("expected error when audit dir cannot be created, got nil")
	}
	if !strings.Contains(err.Error(), "create audit dir") {
		t.Fatalf("expected 'create audit dir' in error, got: %v", err)
	}
}

func TestNewErrorWhenLogFileNotWritable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod 0 has no effect")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	auditDir := filepath.Join(home, ".openpass")
	if err := os.MkdirAll(auditDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Chmod(auditDir, 0o500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	defer os.Chmod(auditDir, 0o700) //nolint:errcheck

	_, err := New("file-fail-agent")
	if err == nil {
		t.Fatal("expected error when log file cannot be opened, got nil")
	}
	if !strings.Contains(err.Error(), "open audit log") {
		t.Fatalf("expected 'open audit log' in error, got: %v", err)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := GetConfig()
	if cfg.MaxFileSize != 100*1024*1024 {
		t.Fatalf("expected default MaxFileSize to be 100MB, got %d", cfg.MaxFileSize)
	}
	if cfg.MaxBackups != 5 {
		t.Fatalf("expected default MaxBackups to be 5, got %d", cfg.MaxBackups)
	}
	if cfg.MaxAgeDays != 30 {
		t.Fatalf("expected default MaxAgeDays to be 30, got %d", cfg.MaxAgeDays)
	}
}

func TestConfigEnvVarMaxSizeMB(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_SIZE_MB", "50")

	ReloadConfig()
	defer ReloadConfig()

	if config.MaxFileSize != 50*1024*1024 {
		t.Fatalf("expected MaxFileSize to be 50MB from env, got %d", config.MaxFileSize)
	}
}

func TestConfigEnvVarMaxBackups(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_BACKUPS", "10")

	ReloadConfig()
	defer ReloadConfig()

	if config.MaxBackups != 10 {
		t.Fatalf("expected MaxBackups to be 10 from env, got %d", config.MaxBackups)
	}
}

func TestConfigEnvVarMaxAgeDays(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_AGE_DAYS", "7")

	ReloadConfig()
	defer ReloadConfig()

	if config.MaxAgeDays != 7 {
		t.Fatalf("expected MaxAgeDays to be 7 from env, got %d", config.MaxAgeDays)
	}
}

func TestConfigEnvVarInvalidValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_SIZE_MB", "invalid")
	t.Setenv("OPENPASS_AUDIT_MAX_BACKUPS", "negative")
	t.Setenv("OPENPASS_AUDIT_MAX_AGE_DAYS", "zero")

	ReloadConfig()
	defer ReloadConfig()

	// Should fall back to defaults
	if config.MaxFileSize != 100*1024*1024 {
		t.Fatalf("expected MaxFileSize to fallback to default, got %d", config.MaxFileSize)
	}
	if config.MaxBackups != 5 {
		t.Fatalf("expected MaxBackups to fallback to default, got %d", config.MaxBackups)
	}
	if config.MaxAgeDays != 30 {
		t.Fatalf("expected MaxAgeDays to fallback to default, got %d", config.MaxAgeDays)
	}
}

func TestRotateIfNeededSizeLimit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_SIZE_MB", "1")

	ReloadConfig()
	defer ReloadConfig()

	logger, err := New("size-rotate-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write until we exceed 1MB
	data := strings.Repeat("x", 1024*1024) // 1MB of data
	for i := 0; i < 2; i++ {
		logger.LogEntry(LogEntry{
			Agent:  "test",
			Action: "test",
			Path:   data,
			OK:     true,
		})
	}

	// Force rotation check
	if err := logger.rotateIfNeeded(); err != nil {
		t.Fatalf("rotateIfNeeded() error = %v", err)
	}

	// Verify rotated file exists
	auditDir := filepath.Join(home, ".openpass")
	pattern := filepath.Join(auditDir, "audit-size-rotate-test.log.rotated.*")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		t.Fatal("expected rotated file to exist after size-based rotation")
	}
}

func TestRotateIfNeededAgeLimit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_AGE_DAYS", "0") // 0 days means immediate

	ReloadConfig()
	defer ReloadConfig()

	logger, err := New("age-rotate-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Set file mod time to yesterday to trigger age-based rotation
	oldTime := time.Now().Add(-48 * time.Hour)
	os.Chtimes(logger.path, oldTime, oldTime) //nolint:errcheck

	// Force rotation check
	if err := logger.rotateIfNeeded(); err != nil {
		t.Fatalf("rotateIfNeeded() error = %v", err)
	}

	// Verify rotated file exists
	auditDir := filepath.Join(home, ".openpass")
	pattern := filepath.Join(auditDir, "audit-age-rotate-test.log.rotated.*")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		t.Fatal("expected rotated file to exist after age-based rotation")
	}
}

func TestEnforceRetentionMaxBackups(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_SIZE_MB", "100")
	t.Setenv("OPENPASS_AUDIT_MAX_BACKUPS", "3")
	t.Setenv("OPENPASS_AUDIT_MAX_AGE_DAYS", "365")

	ReloadConfig()
	defer ReloadConfig()

	logger, err := New("backup-cleanup-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	auditDir := filepath.Join(home, ".openpass")

	// Create 5 rotated files manually
	for i := 0; i < 5; i++ {
		rotatedName := filepath.Join(auditDir, fmt.Sprintf("audit-backup-cleanup-test.log.rotated.%s", time.Now().UTC().Add(time.Duration(i)*time.Second).Format("20060102-150405")))
		if err := os.WriteFile(rotatedName, []byte("test"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		// Stagger times so they're different
		time.Sleep(10 * time.Millisecond)
	}

	// Run retention enforcement
	if err := logger.EnforceRetention(); err != nil {
		t.Fatalf("EnforceRetention() error = %v", err)
	}

	// Check that only 3 backup files remain
	pattern := filepath.Join(auditDir, "audit-backup-cleanup-test.log.rotated.*")
	matches, _ := filepath.Glob(pattern)
	if len(matches) != 3 {
		t.Fatalf("expected 3 backup files after retention policy, got %d", len(matches))
	}
}

func TestEnforceRetentionMaxAge(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_SIZE_MB", "100")
	t.Setenv("OPENPASS_AUDIT_MAX_BACKUPS", "100")
	t.Setenv("OPENPASS_AUDIT_MAX_AGE_DAYS", "0") // 0 days = delete all

	ReloadConfig()
	defer ReloadConfig()

	logger, err := New("age-cleanup-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	auditDir := filepath.Join(home, ".openpass")

	// Create rotated files with old timestamps
	for i := 0; i < 3; i++ {
		rotatedName := filepath.Join(auditDir, fmt.Sprintf("audit-age-cleanup-test.log.rotated.%s", time.Now().UTC().Add(time.Duration(-i-1)*24*time.Hour).Format("20060102-150405")))
		if err := os.WriteFile(rotatedName, []byte("test"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	// Run retention enforcement
	if err := logger.EnforceRetention(); err != nil {
		t.Fatalf("EnforceRetention() error = %v", err)
	}

	// Check that no backup files remain (all are too old)
	pattern := filepath.Join(auditDir, "audit-age-cleanup-test.log.rotated.*")
	matches, _ := filepath.Glob(pattern)
	if len(matches) != 0 {
		t.Fatalf("expected 0 backup files after max age cleanup, got %d", len(matches))
	}
}

func TestEnforceRetentionNoFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := New("no-backups-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Run retention enforcement on clean directory - should not error
	if err := logger.EnforceRetention(); err != nil {
		t.Fatalf("EnforceRetention() error = %v", err)
	}
}

func TestEnforceRetentionPreservesNewest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_BACKUPS", "2")

	ReloadConfig()
	defer ReloadConfig()

	logger, err := New("preserve-newest-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = logger.Close() }()

	auditDir := filepath.Join(home, ".openpass")

	// Create 4 rotated files at different times
	for i := 0; i < 4; i++ {
		rotatedName := filepath.Join(auditDir, fmt.Sprintf("audit-preserve-newest-test.log.rotated.%s", time.Now().UTC().Add(time.Duration(-i)*time.Hour).Format("20060102-150405")))
		if err := os.WriteFile(rotatedName, []byte("test"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Run retention enforcement
	if err := logger.EnforceRetention(); err != nil {
		t.Fatalf("EnforceRetention() error = %v", err)
	}

	// Check that 2 newest files remain
	pattern := filepath.Join(auditDir, "audit-preserve-newest-test.log.rotated.*")
	matches, _ := filepath.Glob(pattern)
	if len(matches) != 2 {
		t.Fatalf("expected 2 backup files (newest), got %d", len(matches))
	}
}

func TestNoLogLossDuringRotation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENPASS_AUDIT_MAX_SIZE_MB", "1")

	ReloadConfig()
	defer ReloadConfig()

	logger, err := New("no-loss-test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Write enough data to exceed 1MB (need > 1MB to trigger rotation)
	largeData := strings.Repeat("x", 100*1024) // 100KB per entry
	for i := 0; i < 15; i++ {
		logger.LogEntry(LogEntry{
			Agent:  "test",
			Action: "test",
			Path:   largeData,
			OK:     true,
		})
	}

	// Verify file is large enough (should exceed 1MB)
	auditDir := filepath.Join(home, ".openpass")
	logFile := filepath.Join(auditDir, "audit-no-loss-test.log")
	info, err := os.Stat(logFile)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size() <= 1*1024*1024 {
		t.Fatalf("expected log file > 1MB to trigger rotation, got %d bytes", info.Size())
	}

	// Trigger rotation
	if rotErr := logger.rotateIfNeeded(); rotErr != nil {
		t.Fatalf("rotateIfNeeded() error = %v", rotErr)
	}

	// Write more entries after rotation
	for i := 0; i < 5; i++ {
		logger.LogEntry(LogEntry{
			Agent:  "test",
			Action: "test2",
			Path:   "test2",
			OK:     true,
		})
	}

	// Check that rotated file exists and has content (no log loss)
	pattern := filepath.Join(auditDir, "audit-no-loss-test.log.rotated.*")
	matches, _ := filepath.Glob(pattern)
	if len(matches) != 1 {
		t.Fatalf("expected 1 rotated file, got %d", len(matches))
	}

	rotatedContent, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	linesRotated := strings.Split(strings.TrimSpace(string(rotatedContent)), "\n")
	if len(linesRotated) != 15 {
		t.Fatalf("expected 15 lines in rotated file, got %d", len(linesRotated))
	}

	// Verify current log file has new entries
	contentAfter, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	linesAfter := strings.Split(strings.TrimSpace(string(contentAfter)), "\n")
	if len(linesAfter) != 5 {
		t.Fatalf("expected 5 lines in current file after rotation, got %d", len(linesAfter))
	}

	defer func() { _ = logger.Close() }()
}
