package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const indexFileName = ".index"

// EntryIndex tracks vault entry metadata for fast listing without filesystem traversal.
type EntryIndex struct {
	Entries []string  `json:"entries"`
	ModTime time.Time `json:"mod_time"`
	mu      sync.RWMutex
}

// indexPath returns the path to the index file within the vault directory.
func indexPath(vaultDir string) string {
	return filepath.Join(vaultDir, indexFileName)
}

// BuildIndex creates a new EntryIndex by scanning the vault directory.
// It uses ListFast for efficient scanning and records the current time.
func BuildIndex(vaultDir string) (*EntryIndex, error) {
	entries, err := ListFast(vaultDir, "")
	if err != nil {
		return nil, fmt.Errorf("build index: %w", err)
	}
	return &EntryIndex{
		Entries: entries,
		ModTime: time.Now().UTC(),
	}, nil
}

// Save writes the index to disk in the vault directory.
func (idx *EntryIndex) Save(vaultDir string) error {
	idx.mu.RLock()
	data, err := json.Marshal(idx)
	idx.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	path := indexPath(vaultDir)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

// LoadIndex reads an existing index from the vault directory.
// Returns nil if the index does not exist.
func LoadIndex(vaultDir string) (*EntryIndex, error) {
	path := indexPath(vaultDir)
	data, err := os.ReadFile(path) //nosec:G304
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}
	var idx EntryIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return &idx, nil
}

// IsStale returns true if the index is older than maxAge or if the vault
// directory has been modified more recently than the index.
func (idx *EntryIndex) IsStale(vaultDir string, maxAge time.Duration) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.Entries) == 0 {
		return true
	}

	if time.Since(idx.ModTime) > maxAge {
		return true
	}

	// Check if vault directory has newer entries
	entriesDirPath := entriesDir(vaultDir)
	info, err := os.Stat(entriesDirPath)
	if err == nil && info.ModTime().After(idx.ModTime) {
		return true
	}

	// Also check legacy root dir
	info, err = os.Stat(vaultDir)
	if err == nil && info.ModTime().After(idx.ModTime) {
		return true
	}

	return false
}

// Filter returns entries matching the given prefix.
func (idx *EntryIndex) Filter(prefix string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if prefix == "" {
		result := make([]string, len(idx.Entries))
		copy(result, idx.Entries)
		return result
	}

	var result []string
	for _, entry := range idx.Entries {
		if strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return result
}

// Update rebuilds the index from the vault directory.
func (idx *EntryIndex) Update(vaultDir string) error {
	entries, err := ListFast(vaultDir, "")
	if err != nil {
		return fmt.Errorf("update index: %w", err)
	}
	sort.Strings(entries)

	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.Entries = entries
	idx.ModTime = time.Now().UTC()
	return nil
}
