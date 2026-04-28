//go:build !cgo

package session

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// memoryKeyring provides an in-memory session cache for builds without CGO.
// Sessions are encrypted at rest using AES-256-GCM and stored in process memory
// only. All sessions are lost when the process exits.
type memoryKeyring struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func vaultDirFromService(service string) string {
	return strings.TrimPrefix(service, "openpass:")
}

func (m *memoryKeyring) Set(service, account, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.store == nil {
		m.store = make(map[string][]byte)
	}

	var sess storedSession
	if err := json.Unmarshal([]byte(value), &sess); err != nil {
		return fmt.Errorf("unmarshal session: %w", err)
	}

	if sess.Passphrase != "" {
		enc, nonce, err := encryptPassphrase(sess.Passphrase, vaultDirFromService(service))
		if err != nil {
			return fmt.Errorf("encrypt passphrase: %w", err)
		}
		sess.EncryptedPassphrase = enc
		sess.Nonce = nonce
		sess.Passphrase = ""
	}

	payload, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	key := service + "|" + account

	if old, ok := m.store[key]; ok {
		for i := range old {
			old[i] = 0
		}
	}

	m.store[key] = append([]byte(nil), payload...)

	return nil
}

func (m *memoryKeyring) Get(service, account string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.store == nil {
		return "", fmt.Errorf("not found")
	}

	key := service + "|" + account
	payload, ok := m.store[key]
	if !ok {
		return "", fmt.Errorf("not found")
	}

	var sess storedSession
	if err := json.Unmarshal(payload, &sess); err != nil {
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}

	if sess.TTL <= 0 {
		for i := range payload {
			payload[i] = 0
		}
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}

	lastActivity := sess.LastAccess
	if lastActivity.IsZero() {
		lastActivity = sess.SavedAt
	}
	if time.Since(lastActivity) > time.Duration(sess.TTL) {
		for i := range payload {
			payload[i] = 0
		}
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}

	if sess.EncryptedPassphrase != "" && sess.Nonce != "" {
		if _, err := decryptPassphrase(sess.EncryptedPassphrase, sess.Nonce, vaultDirFromService(service)); err != nil {
			for i := range payload {
				payload[i] = 0
			}
			delete(m.store, key)
			return "", fmt.Errorf("not found")
		}
	} else if sess.Passphrase == "" {
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}

	sess.LastAccess = time.Now().UTC()
	newPayload, err := json.Marshal(sess)
	if err != nil {
		return "", fmt.Errorf("not found")
	}

	for i := range payload {
		payload[i] = 0
	}
	m.store[key] = append([]byte(nil), newPayload...)

	return string(newPayload), nil
}

func (m *memoryKeyring) Delete(service, account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.store == nil {
		return nil
	}

	key := service + "|" + account
	if payload, ok := m.store[key]; ok {
		for i := range payload {
			payload[i] = 0
		}
		delete(m.store, key)
	}

	return nil
}

var memoryFallbackActive bool

func init() {
	mk := &memoryKeyring{}
	keyringSet = mk.Set
	keyringGet = mk.Get
	keyringDelete = mk.Delete
	memoryFallbackActive = true
	log.Println("WARNING: CGO disabled. Using memory-only session cache (session will clear on process exit).")
}
