//go:build windows

package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const lockFileName = ".lock"

// DefaultLockTimeout is the default timeout for acquiring a write lock.
const DefaultLockTimeout = 30 * time.Second

// AcquireWriteLock opens (or creates) the vault lock file.
//
// TODO: Implement actual locking using LockFileEx from golang.org/x/sys/windows.
// For now, this is a no-op stub that opens the lock file without holding an
// exclusive lock, meaning concurrent vault write operations on Windows are
// NOT serialized. See https://github.com/danieljustus/OpenPass/issues.
func AcquireWriteLock(vaultDir string, timeout time.Duration) (*os.File, error) {
	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}

	lockPath := filepath.Join(vaultDir, lockFileName)

	if err := os.MkdirAll(vaultDir, 0o700); err != nil {
		return nil, fmt.Errorf("create vault directory for lock: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	// No actual locking implemented for Windows yet.
	// LockFileEx would need:
	//   windows.CreateFile with FILE_FLAG_OVERLAPPED
	//   windows.LockFileEx with LOCKFILE_EXCLUSIVE_LOCK

	return f, nil
}

// ReleaseLock closes the lock file.
// No actual unlocking is performed since AcquireWriteLock is a no-op on Windows.
func ReleaseLock(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}
	return lockFile.Close()
}
