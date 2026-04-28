package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// memoryKeyring stores encrypted sessions in process memory only.
type memoryKeyring struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func vaultDirFromService(service string) string {
	return strings.TrimPrefix(service, "openpass:")
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
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
		zeroBytes(old)
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
		zeroBytes(payload)
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}

	lastActivity := sess.LastAccess
	if lastActivity.IsZero() {
		lastActivity = sess.SavedAt
	}
	if time.Since(lastActivity) > time.Duration(sess.TTL) {
		zeroBytes(payload)
		delete(m.store, key)
		return "", fmt.Errorf("not found")
	}

	if sess.EncryptedPassphrase != "" && sess.Nonce != "" {
		if _, err := decryptPassphrase(sess.EncryptedPassphrase, sess.Nonce, vaultDirFromService(service)); err != nil {
			zeroBytes(payload)
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

	zeroBytes(payload)
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
		zeroBytes(payload)
		delete(m.store, key)
	}

	return nil
}
