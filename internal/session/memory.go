//go:build freebsd

package session

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// memoryKeyring provides an in-memory session cache for FreeBSD builds
// where CGO is disabled and OS keyring integration is unavailable.
// Sessions are stored in process memory only and are lost on exit.
type memoryKeyring struct {
	mu    sync.Mutex
	store map[string]string
}

func (m *memoryKeyring) Set(service, account, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.store == nil {
		m.store = make(map[string]string)
	}
	m.store[service+"|"+account] = value
	return nil
}

func (m *memoryKeyring) Get(service, account string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.store == nil {
		return "", fmt.Errorf("not found")
	}

	key := service + "|" + account
	raw, ok := m.store[key]
	if !ok {
		return "", fmt.Errorf("not found")
	}

	var sess storedSession
	if err := json.Unmarshal([]byte(raw), &sess); err != nil {
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}
	if sess.TTL <= 0 {
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}

	lastActivity := sess.LastAccess
	if lastActivity.IsZero() {
		lastActivity = sess.SavedAt
	}
	if time.Since(lastActivity) > time.Duration(sess.TTL) {
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}

	return raw, nil
}

func (m *memoryKeyring) Delete(service, account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.store != nil {
		delete(m.store, service+"|"+account)
	}
	return nil
}

func init() {
	mk := &memoryKeyring{}
	keyringSet = mk.Set
	keyringGet = mk.Get
	keyringDelete = mk.Delete
	memoryFallbackActive = true
	log.Println("WARNING: FreeBSD without CGO detected. Using memory-only session cache (session will clear on process exit).")
}
