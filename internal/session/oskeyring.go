//go:build cgo

package session

import "github.com/zalando/go-keyring"

func init() {
	keyringSet = keyring.Set
	keyringGet = keyring.Get
	keyringDelete = keyring.Delete
}
