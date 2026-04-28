//go:build !cgo

package session

import (
	"log"
)

var memoryFallbackActive bool

func init() {
	mk := &memoryKeyring{}
	keyringSet = mk.Set
	keyringGet = mk.Get
	keyringDelete = mk.Delete
	memoryFallbackActive = true
	log.Println("WARNING: CGO disabled. Using memory-only session cache (session will clear on process exit).")
}
