// Package vaultsvc provides a high-level service layer for vault operations.
package vaultsvc

import (
	"errors"
	"fmt"
	"os"
	"time"

	"filippo.io/age"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

// Service provides high-level vault operations that encapsulate the full
// lifecycle: vault-open → decrypt → operation → encrypt → auto-commit.
type Service struct {
	vault *vaultpkg.Vault
}

// New creates a new vault service for the given vault.
func New(v *vaultpkg.Vault) *Service {
	return &Service{vault: v}
}

// Vault returns the underlying vault instance.
func (s *Service) Vault() *vaultpkg.Vault {
	return s.vault
}

// GetField reads an entry and returns the value of the specified field.
// If field is empty, returns the full entry data map.
// Supports path.field syntax for nested field access.
func (s *Service) GetField(path, field string) (any, error) {
	entry, err := vaultpkg.ReadEntry(s.vault.Dir, path, s.vault.Identity)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, NewError(ErrNotFound, fmt.Sprintf("entry not found: %s", path), err)
		}
		return nil, NewError(ErrReadFailed, fmt.Sprintf("cannot read entry %s: %v", path, err), err)
	}

	if field == "" {
		return entry.Data, nil
	}

	value, ok := entry.Data[field]
	if !ok {
		return nil, NewError(ErrFieldNotFound, fmt.Sprintf("field not found: %s.%s", path, field), nil)
	}

	return value, nil
}

// SetField sets a single field on an entry, creating the entry if it doesn't exist.
// Uses multi-recipient encryption if recipients are configured.
func (s *Service) SetField(path, field string, value any) error {
	data := map[string]any{field: value}
	return s.setEntry(path, data)
}

// SetFields sets multiple fields on an entry, creating the entry if it doesn't exist.
func (s *Service) SetFields(path string, data map[string]any) error {
	return s.setEntry(path, data)
}

// setEntry is the internal upsert implementation shared by SetField and SetFields.
func (s *Service) setEntry(path string, data map[string]any) error {
	existing, readErr := vaultpkg.ReadEntry(s.vault.Dir, path, s.vault.Identity)
	if readErr == nil && existing != nil {
		// Entry exists — merge new data into it
		if _, err := vaultpkg.MergeEntryWithRecipients(s.vault.Dir, path, data, s.vault.Identity); err != nil {
			return NewError(ErrWriteFailed, fmt.Sprintf("cannot update entry %s: %v", path, err), err)
		}
	} else if errors.Is(readErr, os.ErrNotExist) {
		// New entry
		entry := &vaultpkg.Entry{
			Data: data,
			Metadata: vaultpkg.EntryMetadata{
				Created: time.Now().UTC(),
				Updated: time.Now().UTC(),
				Version: 0,
			},
		}
		if err := vaultpkg.WriteEntryWithRecipients(s.vault.Dir, path, entry, s.vault.Identity); err != nil {
			return NewError(ErrWriteFailed, fmt.Sprintf("cannot create entry %s: %v", path, err), err)
		}
	} else {
		return NewError(ErrReadFailed, fmt.Sprintf("cannot read entry %s: %v", path, readErr), readErr)
	}

	// Auto-commit failure is a warning, not an error.
	if err := s.vault.AutoCommit(fmt.Sprintf("Update %s", path)); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: auto-commit failed: %v\n", err)
	}
	return nil
}

// Delete removes an entry from the vault.
func (s *Service) Delete(path string) error {
	if err := vaultpkg.DeleteEntry(s.vault.Dir, path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewError(ErrNotFound, fmt.Sprintf("entry not found: %s", path), err)
		}
		return NewError(ErrWriteFailed, fmt.Sprintf("cannot delete entry %s: %v", path, err), err)
	}

	// Auto-commit failure is a warning, not an error.
	if err := s.vault.AutoCommit(fmt.Sprintf("Delete %s", path)); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: auto-commit failed: %v\n", err)
	}
	return nil
}

// List returns all entry paths, optionally filtered by prefix.
func (s *Service) List(prefix string) ([]string, error) {
	entries, err := vaultpkg.List(s.vault.Dir, prefix)
	if err != nil {
		return nil, NewError(ErrReadFailed, fmt.Sprintf("cannot list entries: %v", err), err)
	}
	return entries, nil
}

// FindOptions configures search behavior.
type FindOptions struct {
	// MaxWorkers controls parallel decryption. <= 0 uses default (CPU count, capped at 4).
	MaxWorkers int
	// ScopeFilter, if non-nil, restricts search to matching paths before decryption.
	ScopeFilter func(path string) bool
}

// Find searches for entries matching the given query.
func (s *Service) Find(query string, opts FindOptions) ([]vaultpkg.Match, error) {
	workers := opts.MaxWorkers
	if workers <= 0 {
		workers = 4 // Default cap to avoid memory exhaustion
	}

	matches, err := vaultpkg.FindWithOptions(s.vault.Dir, query, vaultpkg.FindOptions{
		MaxWorkers:  workers,
		ScopeFilter: opts.ScopeFilter,
	})
	if err != nil {
		return nil, NewError(ErrReadFailed, fmt.Sprintf("search failed: %v", err), err)
	}

	return matches, nil
}

// GetEntry returns the full entry for the given path.
func (s *Service) GetEntry(path string) (*vaultpkg.Entry, error) {
	entry, err := vaultpkg.ReadEntry(s.vault.Dir, path, s.vault.Identity)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, NewError(ErrNotFound, fmt.Sprintf("entry not found: %s", path), err)
		}
		return nil, NewError(ErrReadFailed, fmt.Sprintf("cannot read entry %s: %v", path, err), err)
	}
	return entry, nil
}

// WriteEntry writes a complete entry to the vault.
func (s *Service) WriteEntry(path string, entry *vaultpkg.Entry) error {
	if err := vaultpkg.WriteEntryWithRecipients(s.vault.Dir, path, entry, s.vault.Identity); err != nil {
		return NewError(ErrWriteFailed, fmt.Sprintf("cannot write entry %s: %v", path, err), err)
	}

	// Auto-commit failure is a warning, not an error.
	if err := s.vault.AutoCommit(fmt.Sprintf("Update %s", path)); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: auto-commit failed: %v\n", err)
	}
	return nil
}

// GetIdentity returns the vault's identity for encryption/decryption operations.
func (s *Service) GetIdentity() *age.X25519Identity {
	return s.vault.Identity
}

// GetDir returns the vault directory path.
func (s *Service) GetDir() string {
	return s.vault.Dir
}
