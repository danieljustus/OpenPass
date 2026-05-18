// Package quotas provides process-safe per-tool quota counting using file locking.
//
// QuotaCounter stores per-tool usage counters in a JSON file under the vault
// directory (e.g., <vaultDir>/.quotas.json). All read and write operations
// are serialized via exclusive file locks (flock on Unix, LockFileEx on
// Windows), making the counter safe for concurrent access across processes
// within the same vault.
//
// Usage:
//
//	counter, err := quotas.New("/path/to/vault")
//	if err != nil { ... }
//	defer counter.Close()
//
//	current, err := counter.Increment("read_entry")
//	ok, current := counter.Check("read_entry", 100)
//	counter.Reset()
package quotas

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const quotaFileName = ".quotas.json"

// QuotaCounter provides process-safe per-tool quota counting.
//
// Each QuotaCounter opens and holds the quota JSON file for its lifetime.
// All read-modify-write operations are guarded by an in-process sync.Mutex
// plus an exclusive OS-level file lock (flock / LockFileEx). The OS lock
// ensures cross-process safety; the in-process mutex is required because
// repeated flock calls on the same file descriptor from the same process
// share the same open file description and therefore do not block.
type QuotaCounter struct {
	vaultDir string
	filePath string
	file     *os.File

	opMu sync.Mutex // serializes all in-process operations
}

// quotaData is the on-disk JSON layout.
type quotaData struct {
	Counters map[string]int `json:"counters"`
}

// New opens (or creates) the quota file under vaultDir and returns a
// QuotaCounter ready for use. The caller must call Close when done.
func New(vaultDir string) (*QuotaCounter, error) {
	if err := os.MkdirAll(vaultDir, 0o700); err != nil {
		return nil, fmt.Errorf("create vault directory for quotas: %w", err)
	}

	filePath := filepath.Join(vaultDir, quotaFileName)

	f, err := openQuotaFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("open quota file: %w", err)
	}

	return &QuotaCounter{
		vaultDir: vaultDir,
		filePath: filePath,
		file:     f,
	}, nil
}

// Close releases the quota file handle. The counter must not be used after
// calling Close.
func (qc *QuotaCounter) Close() error {
	qc.opMu.Lock()
	defer qc.opMu.Unlock()

	if qc.file == nil {
		return nil
	}
	err := qc.file.Close()
	qc.file = nil
	return err
}

// Increment increases the counter for toolName by 1 and returns the new
// value. It returns an error if the quota file cannot be locked, read, or
// written.
func (qc *QuotaCounter) Increment(toolName string) (int, error) {
	qc.opMu.Lock()
	defer qc.opMu.Unlock()

	if qc.file == nil {
		return 0, fmt.Errorf("quota counter is closed")
	}

	if err := qc.lock(); err != nil {
		return 0, fmt.Errorf("acquire quota lock: %w", err)
	}
	defer func() { _ = qc.unlock() }()

	data, err := qc.read()
	if err != nil {
		return 0, fmt.Errorf("read quota data: %w", err)
	}

	data.Counters[toolName]++
	current := data.Counters[toolName]

	if err := qc.write(data); err != nil {
		return 0, fmt.Errorf("write quota data: %w", err)
	}

	return current, nil
}

// Check returns (ok, current) where ok is true when the current count for
// toolName is strictly less than limit. If limit is <= 0 it always returns
// false. On any I/O error Check returns (false, 0).
//
// Check acquires the full lock because it must read the latest persisted
// state, which may be concurrently modified by another process.
func (qc *QuotaCounter) Check(toolName string, limit int) (ok bool, current int) {
	if limit <= 0 {
		return false, 0
	}
	qc.opMu.Lock()
	defer qc.opMu.Unlock()

	if qc.file == nil {
		return false, 0
	}

	if err := qc.lock(); err != nil {
		return false, 0
	}
	defer func() { _ = qc.unlock() }()

	data, err := qc.read()
	if err != nil {
		return false, 0
	}

	current = data.Counters[toolName]
	return current < limit, current
}

// Reset zeroes all counters.
func (qc *QuotaCounter) Reset() {
	qc.opMu.Lock()
	defer qc.opMu.Unlock()

	if qc.file == nil {
		return
	}

	if err := qc.lock(); err != nil {
		return
	}
	defer func() { _ = qc.unlock() }()

	_ = qc.write(&quotaData{Counters: make(map[string]int)})
}

// read reads and unmarshals the quota file. The caller must hold the lock.
func (qc *QuotaCounter) read() (*quotaData, error) {
	if _, err := qc.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	raw, err := io.ReadAll(qc.file)
	if err != nil {
		return nil, err
	}

	if len(raw) == 0 {
		return &quotaData{Counters: make(map[string]int)}, nil
	}

	var data quotaData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parse quota file: %w", err)
	}

	if data.Counters == nil {
		data.Counters = make(map[string]int)
	}

	return &data, nil
}

// write marshals and writes the quota file, replacing its contents entirely.
// The caller must hold the lock.
func (qc *QuotaCounter) write(data *quotaData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := qc.file.Truncate(0); err != nil {
		return err
	}

	if _, err := qc.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if _, err := qc.file.Write(raw); err != nil {
		return err
	}

	// Sync to disk while we still hold the lock, ensuring the written state
	// is durable before another process can read it.
	return qc.file.Sync()
}
