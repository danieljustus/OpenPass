//go:build windows

package quotas

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// openQuotaFile opens (or creates) the quota file for read/write with
// FILE_SHARE_READ|FILE_SHARE_WRITE so that concurrent opens from other
// processes succeed, which is required for LockFileEx to work across
// process boundaries.
func openQuotaFile(path string) (*os.File, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("convert quota path: %w", err)
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateFile quota file: %w", err)
	}

	return os.NewFile(uintptr(handle), path), nil
}

// lock acquires an exclusive byte-range lock (LockFileEx) on the quota file.
func (qc *QuotaCounter) lock() error {
	var ol windows.Overlapped
	if err := windows.LockFileEx(
		windows.Handle(qc.file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		1, 0, // lock 1 byte
		&ol,
	); err != nil {
		return fmt.Errorf("LockFileEx: %w", err)
	}
	return nil
}

// unlock releases the byte-range lock (UnlockFileEx) on the quota file.
func (qc *QuotaCounter) unlock() error {
	var ol windows.Overlapped
	if err := windows.UnlockFileEx(
		windows.Handle(qc.file.Fd()),
		0,
		1, 0, // must match the byte range used in lock()
		&ol,
	); err != nil {
		return fmt.Errorf("UnlockFileEx: %w", err)
	}
	return nil
}
