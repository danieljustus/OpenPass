//go:build !windows

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const lockFileName = ".lock"

// DefaultLockTimeout is the default timeout for acquiring a write lock.
const DefaultLockTimeout = 30 * time.Second

// lockFileRetryInterval is the interval between retries when acquiring a lock.
const lockFileRetryInterval = 50 * time.Millisecond

// AcquireWriteLock opens (or creates) the vault lock file and acquires an
// exclusive flock on it. It retries with polling until the timeout expires
// and returns an error if the lock cannot be acquired in time.
func AcquireWriteLock(vaultDir string, timeout time.Duration) (*os.File, error) {
	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}

	lockPath := filepath.Join(vaultDir, lockFileName)

	// Ensure vaultDir exists
	if err := os.MkdirAll(vaultDir, 0o700); err != nil {
		return nil, fmt.Errorf("create vault directory for lock: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600) // #nosec G304 — lockPath is filepath.Join(vaultDir, hardcoded ".lock"); vaultDir is trusted config from caller.
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return f, nil
		}

		if err != syscall.EWOULDBLOCK {
			_ = f.Close()
			return nil, fmt.Errorf("flock error: %w", err)
		}

		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("vault is currently locked by another process, try again in a moment")
		}

		time.Sleep(lockFileRetryInterval)
	}
}

// ReleaseLock releases a file lock and closes the lock file.
func ReleaseLock(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN); err != nil {
		_ = lockFile.Close()
		return fmt.Errorf("unlock error: %w", err)
	}

	return lockFile.Close()
}
