//go:build windows

package crypto

import (
	"unsafe"
)

// SecureString creates a string backed by the original byte slice without
// allocating GC-heap copies. On Windows, mmap/mlock are not available, so this
// always uses the fallback: the returned string shares memory with the input
// slice.
//
// The returned cleanup function MUST be called to wipe the original buffer.
// After cleanup, the string's backing memory is zeroed and must not be accessed.
func SecureString(data []byte) (string, func()) {
	if len(data) == 0 {
		return "", func() {}
	}

	s := unsafe.String(unsafe.SliceData(data), len(data))
	cleanup := func() {
		Wipe(data)
	}
	return s, cleanup
}
