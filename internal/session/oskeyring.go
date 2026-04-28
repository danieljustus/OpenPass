//go:build cgo

package session

import (
	"log"
	"sync"

	"github.com/zalando/go-keyring"
)

var (
	fallbackActive bool
	fallbackMu     sync.RWMutex
	fallback       *memoryKeyring
)

func getFallback() *memoryKeyring {
	fallbackMu.Lock()
	defer fallbackMu.Unlock()

	if fallback == nil {
		fallback = &memoryKeyring{}
		log.Println("WARNING: OS keyring unavailable. Using memory-only session cache (session will clear on process exit).")
	}
	fallbackActive = true
	return fallback
}

func isFallbackActive() bool {
	fallbackMu.RLock()
	defer fallbackMu.RUnlock()
	return fallbackActive
}

func setWithFallback(service, account, value string) error {
	if isFallbackActive() {
		return getFallback().Set(service, account, value)
	}

	if err := keyring.Set(service, account, value); err != nil {
		return getFallback().Set(service, account, value)
	}
	return nil
}

func getWithFallback(service, account string) (string, error) {
	if isFallbackActive() {
		return getFallback().Get(service, account)
	}

	val, err := keyring.Get(service, account)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", err
		}
		return getFallback().Get(service, account)
	}
	return val, nil
}

func deleteWithFallback(service, account string) error {
	if isFallbackActive() {
		return getFallback().Delete(service, account)
	}

	if err := keyring.Delete(service, account); err != nil {
		if err == keyring.ErrNotFound {
			return nil
		}
		return getFallback().Delete(service, account)
	}
	return nil
}

func init() {
	keyringSet = setWithFallback
	keyringGet = getWithFallback
	keyringDelete = deleteWithFallback
}
