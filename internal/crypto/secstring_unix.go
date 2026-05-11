//go:build !windows

package crypto

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// SecureString creates a string backed by locked, non-swappable anonymous
// memory using mmap + mlock, avoiding GC-heap-allocated copies of sensitive
// passphrase data.
//
// The returned cleanup function MUST be called to zero, unlock, and release
// the locked memory. If mmap or mlock fails (e.g., resource limits), SecureString
// falls back to creating a string backed by the original buffer, and the
// cleanup function wipes that buffer instead.
//
// After cleanup, the returned string's backing memory is released and must not
// be accessed. On the mmap+mlock path, the original input slice is not modified
// by cleanup; the caller should continue to wipe it explicitly.
func SecureString(data []byte) (string, func()) {
	if len(data) == 0 {
		return "", func() {}
	}

	buf, err := unix.Mmap(-1, 0, len(data), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err == nil {
		if err := unix.Mlock(buf); err != nil {
			// Mmap succeeded but mlock failed — fall through.
			_ = unix.Munmap(buf)
		} else {
			copy(buf, data)
			s := unsafe.String(&buf[0], len(data))
			cleanup := func() {
				for i := range buf {
					buf[i] = 0
				}
				_ = unix.Munlock(buf)
				_ = unix.Munmap(buf)
				Wipe(data)
			}
			return s, cleanup
		}
	}

	s := unsafe.String(unsafe.SliceData(data), len(data))
	cleanup := func() {
		Wipe(data)
	}
	return s, cleanup
}
