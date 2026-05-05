package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"

	vaultcrypto "github.com/danieljustus/OpenPass/internal/crypto"
	"github.com/danieljustus/OpenPass/internal/metrics"
	"github.com/danieljustus/OpenPass/internal/pathutil"
)

// Entry represents a vault entry with flexible data storage using map[string]any.
type Entry struct {
	Data     map[string]any `json:"data"`
	Metadata EntryMetadata  `json:"meta"`
}

// EntryMetadata contains metadata about an entry
type EntryMetadata struct {
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Version int       `json:"version"`
}

// MarshalJSON implements custom JSON marshaling for Entry
func (e Entry) MarshalJSON() ([]byte, error) {
	type alias Entry
	return json.Marshal(alias(e))
}

// UnmarshalJSON implements custom JSON unmarshaling for Entry
func (e *Entry) UnmarshalJSON(data []byte) error {
	type alias Entry
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*e = Entry(v)
	if e.Data == nil {
		e.Data = map[string]any{}
	}
	return nil
}

// validateEntryPath ensures the entry path stays within the vault directory.
// Returns an error if path traversal is detected.
func validateEntryPath(vaultDir, path string) error {
	if err := validateRawEntryPath(path); err != nil {
		return err
	}

	filePath := entryFilePath(vaultDir, path)
	cleanPath := filepath.Clean(filePath)
	entriesDirClean := filepath.Clean(entriesDir(vaultDir))
	if !strings.HasPrefix(cleanPath, entriesDirClean+string(filepath.Separator)) && cleanPath != entriesDirClean {
		return fmt.Errorf("entry path %q escapes vault directory", path)
	}
	return nil
}

func validateRawEntryPath(path string) error {
	path = strings.TrimSpace(path)

	// Use centralized path validation from pathutil
	if err := pathutil.ValidatePath(path); err != nil {
		return fmt.Errorf("entry path %q: %w", path, err)
	}

	// Additional entry-specific validation: reject "." segments
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, segment := range strings.Split(normalized, "/") {
		if segment == "." {
			return fmt.Errorf("entry path %q contains invalid path segment \".\"", path)
		}
	}

	return nil
}

func validateLegacyEntryPath(vaultDir, path string) error {
	if err := validateRawEntryPath(path); err != nil {
		return err
	}

	filePath := legacyEntryFilePath(vaultDir, path)
	cleanPath := filepath.Clean(filePath)
	vaultDirClean := filepath.Clean(vaultDir)
	if !strings.HasPrefix(cleanPath, vaultDirClean+string(filepath.Separator)) || cleanPath == filepath.Join(vaultDirClean, "identity.age") {
		return fmt.Errorf("legacy entry path %q escapes vault entry namespace", path)
	}
	return nil
}

// ReadEntry reads and decrypts an entry from the vault
func ReadEntry(vaultDir, path string, identity *age.X25519Identity) (*Entry, error) {
	if identity == nil {
		return nil, errors.New("nil identity")
	}
	rememberSearchIdentity(identity)

	if err := validateEntryPath(vaultDir, path); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(entryFilePath(vaultDir, path))
	if os.IsNotExist(err) && canUseLegacyEntryPath(path) {
		if legacyErr := validateLegacyEntryPath(vaultDir, path); legacyErr != nil {
			return nil, legacyErr
		}
		raw, err = os.ReadFile(legacyEntryFilePath(vaultDir, path))
	}
	if err != nil {
		return nil, err
	}

	start := time.Now()
	plaintext, err := vaultcrypto.Decrypt(raw, identity)
	metrics.RecordVaultOperationDuration("decrypt", time.Since(start))
	if err != nil {
		return nil, err
	}
	defer vaultcrypto.Wipe(plaintext)

	var entry Entry
	if err := json.Unmarshal(plaintext, &entry); err != nil {
		return nil, err
	}
	if entry.Data == nil {
		entry.Data = map[string]any{}
	}
	return &entry, nil
}

// WriteEntry encrypts and writes an entry to the vault
func WriteEntry(vaultDir, path string, entry *Entry, identity *age.X25519Identity) error {
	if entry == nil {
		return errors.New("nil entry")
	}
	if identity == nil {
		return errors.New("nil identity")
	}
	rememberSearchIdentity(identity)

	if err := validateEntryPath(vaultDir, path); err != nil {
		return err
	}

	now := time.Now().UTC()
	copyEntry := cloneEntry(entry)
	if copyEntry.Metadata.Created.IsZero() {
		copyEntry.Metadata.Created = now
	}
	copyEntry.Metadata.Updated = now
	copyEntry.Metadata.Version++
	if copyEntry.Data == nil {
		copyEntry.Data = map[string]any{}
	}

	plaintext, err := json.Marshal(copyEntry)
	if err != nil {
		return err
	}

	start := time.Now()
	ciphertext, err := vaultcrypto.Encrypt(plaintext, identity.Recipient())
	metrics.RecordVaultOperationDuration("encrypt", time.Since(start))
	if err != nil {
		return err
	}

	filePath := entryFilePath(vaultDir, path)
	if err := SafeMkdirAll(filepath.Dir(filePath), 0o700); err != nil {
		return err
	}
	// Symlink-hardened write: O_NOFOLLOW + fstat verification prevents writing through symlinks
	return SafeWriteFile(filePath, ciphertext, 0o600)
}

// DeleteEntry removes an entry from the vault
func DeleteEntry(vaultDir, path string) error {
	if err := validateEntryPath(vaultDir, path); err != nil {
		return err
	}

	filePath := entryFilePath(vaultDir, path)
	// Symlink-hardened remove: O_NOFOLLOW + fstat verification prevents removing through symlinks
	if err := SafeRemove(filePath); err != nil {
		if !os.IsNotExist(err) || !canUseLegacyEntryPath(path) {
			return err
		}
		if legacyErr := validateLegacyEntryPath(vaultDir, path); legacyErr != nil {
			return legacyErr
		}
		if err := SafeRemove(legacyEntryFilePath(vaultDir, path)); err != nil {
			return err
		}
		return nil
	}

	if canUseLegacyEntryPath(path) {
		if legacyErr := validateLegacyEntryPath(vaultDir, path); legacyErr != nil {
			return legacyErr
		}
		if err := SafeRemove(legacyEntryFilePath(vaultDir, path)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

const entriesDirName = "entries"

func entriesDir(vaultDir string) string {
	return filepath.Join(vaultDir, entriesDirName)
}

func canUseLegacyEntryPath(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return clean != "identity" && clean != entriesDirName && !strings.HasPrefix(clean, entriesDirName+"/")
}

func legacyEntryFilePath(vaultDir, path string) string {
	return filepath.Join(vaultDir, filepath.FromSlash(path)+".age")
}

func migrateLegacyEntries(vaultDir string) error {
	vaultDirClean := filepath.Clean(vaultDir)
	entriesDirClean := filepath.Clean(entriesDir(vaultDir))
	// Symlink-hardened mkdir: validates each component to prevent following symlinks
	if err := SafeMkdirAll(entriesDirClean, 0o700); err != nil {
		return err
	}

	return filepath.Walk(vaultDirClean, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == vaultDirClean {
			return nil
		}
		if info.IsDir() {
			cleanPath := filepath.Clean(path)
			if cleanPath == entriesDirClean || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".age" { //nolint:goconst // file extension literal
			return nil
		}

		rel, err := filepath.Rel(vaultDirClean, path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(rel) == "identity.age" { //nolint:goconst // filename literal
			return nil
		}

		target := filepath.Join(entriesDirClean, rel)
		if _, err := os.Stat(target); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := SafeMkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		return os.Rename(path, target) // #nosec G122 -- both paths are within the user's own vault directory (internally generated by filepath.Walk)
	})
}

// MergeEntry merges partial data into an existing entry
func MergeEntry(vaultDir, path string, partialData map[string]any, identity *age.X25519Identity) (*Entry, error) {
	entry, err := ReadEntry(vaultDir, path, identity)
	if err != nil {
		return nil, err
	}
	if entry.Data == nil {
		entry.Data = map[string]any{}
	}
	mergeMaps(entry.Data, partialData)
	if err := WriteEntry(vaultDir, path, entry, identity); err != nil {
		return nil, err
	}
	return ReadEntry(vaultDir, path, identity)
}

// entryFilePath returns the filesystem path for an entry
func entryFilePath(vaultDir, path string) string {
	return filepath.Join(entriesDir(vaultDir), filepath.FromSlash(path)+".age")
}

// cloneEntry creates a deep copy of an entry
func cloneEntry(entry *Entry) *Entry {
	if entry == nil {
		return nil
	}
	clone := &Entry{Metadata: entry.Metadata}
	if entry.Data != nil {
		if cloned, ok := deepCloneMap(entry.Data).(map[string]any); ok {
			clone.Data = cloned
		}
	}
	return clone
}

// deepCloneMap creates a deep copy of a map
func deepCloneMap(m map[string]any) any {
	clone := make(map[string]any, len(m))
	for k, v := range m {
		clone[k] = deepCloneValue(v)
	}
	return clone
}

// deepCloneValue creates a deep copy of a value
func deepCloneValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return deepCloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = deepCloneValue(typed[i])
		}
		return out
	default:
		return typed
	}
}

// mergeMaps merges source map into destination map
func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		if existing, ok := dst[k]; ok {
			dstMap, dstIsMap := existing.(map[string]any)
			srcMap, srcIsMap := v.(map[string]any)
			if dstIsMap && srcIsMap {
				mergeMaps(dstMap, srcMap)
				dst[k] = dstMap
				continue
			}
		}
		dst[k] = deepCloneValue(v)
	}
}

// ExtractTOTP extracts TOTP configuration from entry data.
// Returns the secret, algorithm, digits, period, and a boolean indicating
// whether a valid TOTP configuration was found.
func ExtractTOTP(data map[string]any) (secret, algorithm string, digits, period int, hasTOTP bool) {
	totpData, ok := data["totp"].(map[string]any)
	if !ok {
		return "", "", 0, 0, false
	}

	secretVal, ok := totpData["secret"].(string)
	if !ok || secretVal == "" {
		return "", "", 0, 0, false
	}

	algorithm = "SHA1"
	if v, ok := totpData["algorithm"].(string); ok && v != "" {
		algorithm = v
	}

	digits = 6
	if v, ok := totpData["digits"].(float64); ok {
		digits = int(v)
	}

	period = 30
	if v, ok := totpData["period"].(float64); ok {
		period = int(v)
	}

	return secretVal, algorithm, digits, period, true
}

// GetField retrieves a field value from the entry's data map.
func (e *Entry) GetField(name string) (any, bool) {
	if e.Data == nil {
		return nil, false
	}
	val, ok := e.Data[name]
	return val, ok
}

// GetEntryMetadata reads only the metadata from an entry without decrypting the full entry.
// This is useful for cache validation where only freshness information is needed.
// Returns the metadata and a boolean indicating if the entry exists.
func GetEntryMetadata(vaultDir, path string, identity *age.X25519Identity) (*EntryMetadata, error) {
	if identity == nil {
		return nil, errors.New("nil identity")
	}

	if err := validateEntryPath(vaultDir, path); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(entryFilePath(vaultDir, path))
	if os.IsNotExist(err) && canUseLegacyEntryPath(path) {
		if legacyErr := validateLegacyEntryPath(vaultDir, path); legacyErr != nil {
			return nil, legacyErr
		}
		raw, err = os.ReadFile(legacyEntryFilePath(vaultDir, path))
	}
	if err != nil {
		return nil, err
	}

	start := time.Now()
	plaintext, err := vaultcrypto.Decrypt(raw, identity)
	metrics.RecordVaultOperationDuration("decrypt", time.Since(start))
	if err != nil {
		return nil, err
	}
	defer vaultcrypto.Wipe(plaintext)

	// Only unmarshal the metadata portion for efficiency
	var entry struct {
		Metadata EntryMetadata `json:"meta"`
	}
	if err := json.Unmarshal(plaintext, &entry); err != nil {
		return nil, err
	}

	return &entry.Metadata, nil
}

// SetField sets a field value in the entry's data map
func (e *Entry) SetField(name string, value any) {
	if e.Data == nil {
		e.Data = make(map[string]any)
	}
	e.Data[name] = value
}

// HasField checks if a field exists in the entry
func (e *Entry) HasField(name string) bool {
	if e.Data == nil {
		return false
	}
	_, ok := e.Data[name]
	return ok
}
