// Package audit provides audit logging for MCP tool calls.
package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultDirName         = ".openpass"
	defaultFileNamePattern = "audit-%s.log"
)

// Environment variable names for audit configuration
const (
	envMaxSizeMB  = "OPENPASS_AUDIT_MAX_SIZE_MB"
	envMaxBackups = "OPENPASS_AUDIT_MAX_BACKUPS"
	envMaxAgeDays = "OPENPASS_AUDIT_MAX_AGE_DAYS"
)

// AuditConfig holds the configuration for audit log rotation.
// These values are set at package initialization from environment variables.
type AuditConfig struct {
	// MaxFileSize is the maximum size of a single audit log file before rotation.
	MaxFileSize int64

	// MaxBackups is the maximum number of rotated backup files to retain.
	MaxBackups int

	// MaxAgeDays is the maximum age of a rotated backup file in days before deletion.
	MaxAgeDays int
}

// config holds the parsed audit configuration.
var config = parseAuditConfig()

// ReloadConfig re-parses configuration from environment variables.
// Exported for testing purposes.
func ReloadConfig() {
	config = parseAuditConfig()
}

func parseAuditConfig() AuditConfig {
	cfg := AuditConfig{
		// Default: 100MB per the task requirement
		MaxFileSize: 100 * 1024 * 1024,
		// Default: 5 backups per the task requirement
		MaxBackups: 5,
		// Default: 30 days per the task requirement
		MaxAgeDays: 30,
	}

	if val := os.Getenv(envMaxSizeMB); val != "" {
		if mb, err := strconv.ParseInt(val, 10, 64); err == nil && mb > 0 {
			cfg.MaxFileSize = mb * 1024 * 1024
		}
	}

	if val := os.Getenv(envMaxBackups); val != "" {
		if backups, err := strconv.Atoi(val); err == nil && backups >= 0 {
			cfg.MaxBackups = backups
		}
	}

	if val := os.Getenv(envMaxAgeDays); val != "" {
		if days, err := strconv.Atoi(val); err == nil && days >= 0 {
			cfg.MaxAgeDays = days
		}
	}

	return cfg
}

// GetConfig returns the current audit configuration.
func GetConfig() AuditConfig {
	return config
}

// HealthStatus represents the health status of audit logging.
type HealthStatus struct {
	LogFilePath     string `json:"log_file_path"`
	LogFileAge      string `json:"log_file_age"`
	LastEntryTime   string `json:"last_entry_time,omitempty"`
	Agent           string `json:"agent"`
	TotalAuditSize  int64  `json:"total_audit_size_bytes"`
	LogFileSize     int64  `json:"log_file_size_bytes"`
	ErrorCount      int    `json:"error_count_last_100"`
	LastEntryOK     *bool  `json:"last_entry_ok,omitempty"`
	OK              bool   `json:"ok"`
	WriteAccessible bool   `json:"write_accessible"`
	NeedsRotation   bool   `json:"needs_rotation"`
	NeedsRetention  bool   `json:"needs_retention"`
}

// ErrorInfo represents a redacted error from audit logs.
type ErrorInfo struct {
	Timestamp string `json:"ts"`
	Action    string `json:"action"`
	Reason    string `json:"reason,omitempty"`
	OK        bool   `json:"ok"`
}

type LogEntry struct {
	Timestamp string `json:"ts"`
	Agent     string `json:"agent"`
	Action    string `json:"action"`
	Path      string `json:"path,omitempty"`
	Field     string `json:"field,omitempty"`
	Transport string `json:"transport,omitempty"`
	Reason    string `json:"reason,omitempty"`
	DurMs     int64  `json:"dur_ms,omitempty"`
	OK        bool   `json:"ok"`
}

type Logger struct {
	agentName string
	path      string
	file      *os.File
	mu        sync.Mutex
}

func New(agentName string) (*Logger, error) {
	if strings.Contains(agentName, "/") || strings.Contains(agentName, "\\") || agentName == ".." || agentName == "." {
		return nil, errors.New("agent name must not contain path separators or traversal patterns")
	}
	if strings.Contains(agentName, "..") {
		return nil, errors.New("agent name must not contain path traversal patterns")
	}

	home := os.Getenv("HOME")
	if home == "" {
		resolved, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		home = resolved
	}

	cleanHome := filepath.Clean(home)
	auditDir := filepath.Join(cleanHome, defaultDirName)
	cleanAuditDir := filepath.Clean(auditDir)
	if !strings.HasPrefix(cleanAuditDir, cleanHome+string(filepath.Separator)) {
		return nil, errors.New("invalid audit directory path")
	}
	if err := os.MkdirAll(cleanAuditDir, 0o700); err != nil {
		return nil, fmt.Errorf("create audit dir: %w", err)
	}

	path := filepath.Join(cleanAuditDir, fmt.Sprintf(defaultFileNamePattern, agentName))
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, cleanAuditDir+string(filepath.Separator)) {
		return nil, errors.New("agent name resulted in path outside audit directory")
	}

	file, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // path is validated above
	if err != nil {
		return nil, fmt.Errorf("open audit log: %w", err)
	}

	l := &Logger{
		file:      file,
		agentName: agentName,
		path:      cleanPath,
	}

	if err := l.rotateIfNeeded(); err != nil {
		_ = l.Close()
		return nil, fmt.Errorf("check rotation: %w", err)
	}

	return l, nil
}

func (l *Logger) LogEntry(entry LogEntry) {
	if l == nil || l.file == nil {
		return
	}

	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')
	if _, err := l.file.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "audit log write failed: %v\n", err)
	}
	if err := l.file.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "audit log sync failed: %v\n", err)
	}
}

func (l *Logger) rotateIfNeeded() error {
	if l == nil || l.file == nil {
		return nil
	}

	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	needsRotation := info.Size() >= config.MaxFileSize

	if !needsRotation {
		age := time.Since(info.ModTime())
		maxFileAge := time.Duration(config.MaxAgeDays) * 24 * time.Hour
		needsRotation = age >= maxFileAge
	}

	if !needsRotation {
		return nil
	}

	_ = l.file.Close()

	rotatePath := l.path + ".rotated." + time.Now().UTC().Format("20060102-150405")
	if err = os.Rename(l.path, rotatePath); err != nil {
		// Try to reopen original file
		l.file, err = os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("rename and reopen: %w", err)
		}
		return fmt.Errorf("rotate log: %w", err)
	}

	l.file, err = os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("reopen after rotation: %w", err)
	}

	return nil
}

// EnforceRetention removes old rotated files based on retention policy:
// - Removes oldest files if count exceeds MaxBackups
// - Removes files older than MaxAgeDays
func (l *Logger) EnforceRetention() error {
	if l == nil {
		return errors.New("logger is nil")
	}

	auditDir := filepath.Dir(l.path)
	pattern := filepath.Join(auditDir, fmt.Sprintf(defaultFileNamePattern, l.agentName)+".rotated.*")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob rotated files: %w", err)
	}

	var rotatedFiles []os.FileInfo
	now := time.Now()
	maxAge := time.Duration(config.MaxAgeDays) * 24 * time.Hour

	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		rotatedFiles = append(rotatedFiles, info)
	}

	if len(rotatedFiles) == 0 {
		return nil
	}

	// Sort by modification time, oldest first
	for i := 0; i < len(rotatedFiles)-1; i++ {
		for j := i + 1; j < len(rotatedFiles); j++ {
			if rotatedFiles[i].ModTime().After(rotatedFiles[j].ModTime()) {
				rotatedFiles[i], rotatedFiles[j] = rotatedFiles[j], rotatedFiles[i]
			}
		}
	}

	// Check max backups policy - keep at most MaxBackups files
	backupsToDelete := len(rotatedFiles) - config.MaxBackups
	if backupsToDelete > 0 {
		for i := 0; i < backupsToDelete; i++ {
			info := rotatedFiles[i]
			path := filepath.Join(auditDir, info.Name())
			if err := os.Remove(path); err != nil {
				continue
			}
		}
		rotatedFiles = rotatedFiles[backupsToDelete:]
	}

	// Check max age policy - remove files older than MaxAgeDays
	for _, info := range rotatedFiles {
		age := now.Sub(info.ModTime())
		if age >= maxAge {
			path := filepath.Join(auditDir, info.Name())
			if err := os.Remove(path); err != nil {
				continue
			}
		}
	}

	return nil
}

// HealthCheck returns the health status of the audit logger.
func (l *Logger) HealthCheck() (*HealthStatus, error) {
	if l == nil || l.file == nil {
		return &HealthStatus{OK: false}, errors.New("logger not initialized")
	}

	status := &HealthStatus{
		OK:          true,
		Agent:       l.agentName,
		LogFilePath: l.path,
	}

	info, err := l.file.Stat()
	if err != nil {
		status.OK = false
		status.WriteAccessible = false
		return status, fmt.Errorf("stat log file: %w", err)
	}

	status.LogFileSize = info.Size()
	status.LogFileAge = time.Since(info.ModTime()).Round(time.Second).String()
	status.WriteAccessible = info.Mode().Perm()&0200 != 0

	maxFileAge := time.Duration(config.MaxAgeDays) * 24 * time.Hour
	if info.Size() >= config.MaxFileSize || status.LogFileAge >= maxFileAge.String() {
		status.NeedsRotation = true
	}

	// Check total audit size across all files in directory
	auditDir := filepath.Dir(l.path)
	pattern := filepath.Join(auditDir, fmt.Sprintf(defaultFileNamePattern, l.agentName)+"*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		// Log error but continue with what we have
		fmt.Fprintf(os.Stderr, "failed to glob audit files: %v\n", err)
	}
	for _, path := range matches {
		if info, statErr := os.Stat(path); statErr == nil {
			status.TotalAuditSize += info.Size()
		}
	}
	if status.TotalAuditSize >= config.MaxFileSize {
		status.NeedsRetention = true
	}

	// Read last 100 entries to get error count and last entry
	last100, err := l.lastNEntries(100)
	if err == nil {
		for _, entry := range last100 {
			if !entry.OK {
				status.ErrorCount++
			}
			if status.LastEntryTime == "" {
				status.LastEntryTime = entry.Timestamp
				status.LastEntryOK = &entry.OK
			}
		}
	}

	return status, nil
}

func (l *Logger) lastNEntries(n int) ([]LogEntry, error) {
	if l == nil || l.file == nil {
		return nil, errors.New("logger not initialized")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	info, err := l.file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := info.Size()
	if fileSize == 0 {
		return nil, nil
	}

	var entries []LogEntry
	reader := io.NewSectionReader(l.file, 0, fileSize)

	scanner := bufio.NewScanner(reader)
	scanner.Split(scanLines)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
			if len(entries) > n {
				entries = entries[len(entries)-n:]
			}
		}
	}

	return entries, nil
}

func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := strings.Index(string(data), "\n"); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// GetErrors returns redacted error entries from the audit log.
func (l *Logger) GetErrors(limit int) ([]ErrorInfo, error) {
	entries, err := l.lastNEntries(limit)
	if err != nil {
		return nil, err
	}

	var errors []ErrorInfo
	for _, entry := range entries {
		if !entry.OK {
			errors = append(errors, ErrorInfo{
				Timestamp: entry.Timestamp,
				Action:    entry.Action,
				Reason:    entry.Reason,
				OK:        entry.OK,
			})
		}
	}

	return errors, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}
