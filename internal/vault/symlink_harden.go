//go:build !windows

package vault

import (
	"os"
	"syscall"
)

func SafeWriteFile(path string, data []byte, perm os.FileMode) error {
	flags := syscall.O_NOFOLLOW | os.O_CREATE | os.O_TRUNC | os.O_WRONLY

	fd, err := syscall.Open(path, flags, uint32(perm))
	if err != nil {
		return &os.PathError{Op: "open", Path: path, Err: err}
	}
	defer func() { _ = syscall.Close(fd) }()

	var stat syscall.Stat_t
	if err = syscall.Fstat(fd, &stat); err != nil {
		return &os.PathError{Op: "fstat", Path: path, Err: err}
	}

	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG {
		return &os.PathError{
			Op:   "open",
			Path: path,
			Err:  syscall.ENOTDIR,
		}
	}

	n, err := syscall.Write(fd, data)
	if err != nil {
		return &os.PathError{Op: "write", Path: path, Err: err}
	}
	if n != len(data) {
		return &os.PathError{Op: "write", Path: path, Err: syscall.ENOSPC}
	}

	return nil
}

func SafeRemove(path string) error {
	flags := syscall.O_NOFOLLOW | syscall.O_RDONLY

	fd, err := syscall.Open(path, flags, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: path, Err: err}
	}
	defer func() { _ = syscall.Close(fd) }()

	var stat syscall.Stat_t
	if err = syscall.Fstat(fd, &stat); err != nil {
		return &os.PathError{Op: "fstat", Path: path, Err: err}
	}

	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG {
		return &os.PathError{
			Op:   "open",
			Path: path,
			Err:  syscall.ENOTDIR,
		}
	}

	if err = syscall.Close(fd); err != nil {
		return &os.PathError{Op: "close", Path: path, Err: err}
	}

	return os.Remove(path)
}

func SafeMkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
