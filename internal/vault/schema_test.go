package vault

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/danieljustus/OpenPass/internal/testutil"
)

func TestNewEntryV2InitializesDefaults(t *testing.T) {
	entry := NewEntryV2()

	if entry == nil {
		t.Fatal("NewEntryV2 returned nil")
	}
	if entry.Tags == nil {
		t.Fatal("Tags should be initialized")
	}
	if len(entry.Tags) != 0 {
		t.Fatalf("Tags should be empty, got %v", entry.Tags)
	}
	if entry.CustomFields == nil {
		t.Fatal("CustomFields should be initialized")
	}
	if len(entry.CustomFields) != 0 {
		t.Fatalf("CustomFields should be empty, got %v", entry.CustomFields)
	}
	if entry.Version != 1 {
		t.Fatalf("Version should be 1, got %d", entry.Version)
	}
	if entry.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}
	if entry.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt should be set")
	}
	if !entry.CreatedAt.Equal(entry.UpdatedAt) {
		t.Fatal("CreatedAt and UpdatedAt should be equal for new entries")
	}
}

func TestEntryV2UpdateTimestamps(t *testing.T) {
	entry := NewEntryV2()
	originalUpdated := entry.UpdatedAt
	originalVersion := entry.Version

	waitForTimeToAdvance := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Millisecond)
		close(waitForTimeToAdvance)
	}()
	<-waitForTimeToAdvance

	entry.UpdateTimestamps()

	if !entry.UpdatedAt.After(originalUpdated) {
		t.Fatal("UpdatedAt should be after original")
	}
	if entry.Version != originalVersion+1 {
		t.Fatalf("Version should increment, got %d, want %d", entry.Version, originalVersion+1)
	}
	if entry.CreatedAt.After(entry.UpdatedAt) {
		t.Fatal("CreatedAt should not be after UpdatedAt")
	}
}

func TestEntryV2AddTag(t *testing.T) {
	entry := NewEntryV2()

	entry.AddTag("work")
	if len(entry.Tags) != 1 || entry.Tags[0] != "work" {
		t.Fatalf("expected [work], got %v", entry.Tags)
	}

	entry.AddTag("personal")
	if len(entry.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(entry.Tags))
	}

	entry.AddTag("work")
	if len(entry.Tags) != 2 {
		t.Fatalf("expected 2 tags after duplicate add, got %d", len(entry.Tags))
	}
}

func TestEntryV2RemoveTag(t *testing.T) {
	entry := NewEntryV2()
	entry.Tags = []string{"work", "personal", "important"}

	entry.RemoveTag("personal")
	if len(entry.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(entry.Tags))
	}
	for _, tag := range entry.Tags {
		if tag == "personal" {
			t.Fatal("personal tag should be removed")
		}
	}

	entry.RemoveTag("nonexistent")
	if len(entry.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(entry.Tags))
	}

	entry.RemoveTag("work")
	entry.RemoveTag("important")
	if len(entry.Tags) != 0 {
		t.Fatalf("expected 0 tags, got %d", len(entry.Tags))
	}
}

func TestEntryV2AddCustomField(t *testing.T) {
	entry := NewEntryV2()

	field1 := CustomField{Name: "pin", Value: "1234", Type: FieldTypeHidden}
	entry.AddCustomField(field1)

	if len(entry.CustomFields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(entry.CustomFields))
	}

	field2 := CustomField{Name: "email", Value: "test@example.com", Type: FieldTypeEmail}
	entry.AddCustomField(field2)

	if len(entry.CustomFields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(entry.CustomFields))
	}

	field1Updated := CustomField{Name: "pin", Value: "5678", Type: FieldTypeHidden}
	entry.AddCustomField(field1Updated)

	if len(entry.CustomFields) != 2 {
		t.Fatalf("expected 2 fields after update, got %d", len(entry.CustomFields))
	}

	found, ok := entry.GetCustomField("pin")
	if !ok {
		t.Fatal("should find pin field")
	}
	if found.Value != "5678" {
		t.Fatalf("expected updated value 5678, got %s", found.Value)
	}
}

func TestEntryV2GetCustomField(t *testing.T) {
	entry := NewEntryV2()
	entry.CustomFields = []CustomField{
		{Name: "pin", Value: "1234", Type: FieldTypeHidden},
		{Name: "email", Value: "test@example.com", Type: FieldTypeEmail},
	}

	field, ok := entry.GetCustomField("pin")
	if !ok {
		t.Fatal("should find pin field")
	}
	if field.Value != "1234" {
		t.Fatalf("expected value 1234, got %s", field.Value)
	}

	_, ok = entry.GetCustomField("nonexistent")
	if ok {
		t.Fatal("should not find nonexistent field")
	}
}

func TestEntryV2RemoveCustomField(t *testing.T) {
	entry := NewEntryV2()
	entry.CustomFields = []CustomField{
		{Name: "pin", Value: "1234", Type: FieldTypeHidden},
		{Name: "email", Value: "test@example.com", Type: FieldTypeEmail},
	}

	entry.RemoveCustomField("pin")
	if len(entry.CustomFields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(entry.CustomFields))
	}

	_, ok := entry.GetCustomField("pin")
	if ok {
		t.Fatal("pin field should be removed")
	}

	entry.RemoveCustomField("nonexistent")
	if len(entry.CustomFields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(entry.CustomFields))
	}
}

func TestEntryV2JSONSerialization(t *testing.T) {
	created := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 3, 30, 10, 5, 0, 0, time.UTC)

	entry := &EntryV2{
		Name:     "Test Entry",
		Username: "testuser",
		Password: "secret123",
		URL:      "https://example.com",
		Notes:    "Test notes",
		Tags:     []string{"work", "test"},
		TOTP: &TOTPConfig{
			Secret:      "JBSWY3DPEHPK3PXP",
			Algorithm:   "SHA1",
			Digits:      6,
			Period:      30,
			Issuer:      "Example",
			AccountName: "testuser",
		},
		CustomFields: []CustomField{
			{Name: "pin", Value: "1234", Type: FieldTypeHidden},
		},
		CreatedAt: created,
		UpdatedAt: updated,
		Version:   5,
	}

	data, err := entry.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got EntryV2
	if err := got.UnmarshalJSON(data); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(got.Name, entry.Name) {
		t.Errorf("Name mismatch: got %v, want %v", got.Name, entry.Name)
	}
	if !reflect.DeepEqual(got.Username, entry.Username) {
		t.Errorf("Username mismatch: got %v, want %v", got.Username, entry.Username)
	}
	if !reflect.DeepEqual(got.Password, entry.Password) {
		t.Errorf("Password mismatch: got %v, want %v", got.Password, entry.Password)
	}
	if !reflect.DeepEqual(got.URL, entry.URL) {
		t.Errorf("URL mismatch: got %v, want %v", got.URL, entry.URL)
	}
	if !reflect.DeepEqual(got.Notes, entry.Notes) {
		t.Errorf("Notes mismatch: got %v, want %v", got.Notes, entry.Notes)
	}
	if !reflect.DeepEqual(got.Tags, entry.Tags) {
		t.Errorf("Tags mismatch: got %v, want %v", got.Tags, entry.Tags)
	}
	if !reflect.DeepEqual(got.Version, entry.Version) {
		t.Errorf("Version mismatch: got %v, want %v", got.Version, entry.Version)
	}
	if !got.CreatedAt.Equal(entry.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", got.CreatedAt, entry.CreatedAt)
	}
	if !got.UpdatedAt.Equal(entry.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: got %v, want %v", got.UpdatedAt, entry.UpdatedAt)
	}

	if got.TOTP == nil {
		t.Fatal("TOTP should not be nil")
	}
	if got.TOTP.Secret != entry.TOTP.Secret {
		t.Errorf("TOTP.Secret mismatch: got %v, want %v", got.TOTP.Secret, entry.TOTP.Secret)
	}
	if got.TOTP.Algorithm != entry.TOTP.Algorithm {
		t.Errorf("TOTP.Algorithm mismatch: got %v, want %v", got.TOTP.Algorithm, entry.TOTP.Algorithm)
	}
	if got.TOTP.Digits != entry.TOTP.Digits {
		t.Errorf("TOTP.Digits mismatch: got %v, want %v", got.TOTP.Digits, entry.TOTP.Digits)
	}
	if got.TOTP.Period != entry.TOTP.Period {
		t.Errorf("TOTP.Period mismatch: got %v, want %v", got.TOTP.Period, entry.TOTP.Period)
	}

	if len(got.CustomFields) != len(entry.CustomFields) {
		t.Errorf("CustomFields length mismatch: got %d, want %d", len(got.CustomFields), len(entry.CustomFields))
	}
}

func TestEntryV2UnmarshalJSONInitializesSlices(t *testing.T) {
	data := []byte(`{"name":"test"}`)

	var entry EntryV2
	if err := entry.UnmarshalJSON(data); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if entry.Tags == nil {
		t.Fatal("Tags should be initialized even with empty JSON")
	}
	if entry.CustomFields == nil {
		t.Fatal("CustomFields should be initialized even with empty JSON")
	}
}

func TestEntryV2ToLegacyEntry(t *testing.T) {
	created := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 3, 30, 10, 5, 0, 0, time.UTC)

	entry := &EntryV2{
		Name:     "Test Entry",
		Username: "testuser",
		Password: "secret123",
		URL:      "https://example.com",
		Notes:    "Test notes",
		Tags:     []string{"work", "test"},
		TOTP: &TOTPConfig{
			Secret:      "JBSWY3DPEHPK3PXP",
			Algorithm:   "SHA256",
			Digits:      8,
			Period:      60,
			Issuer:      "Example",
			AccountName: "testuser",
		},
		CustomFields: []CustomField{
			{Name: "pin", Value: "1234", Type: FieldTypeHidden},
			{Name: "url2", Value: "https://other.com", Type: FieldTypeURL},
		},
		CreatedAt: created,
		UpdatedAt: updated,
		Version:   5,
	}

	legacy := entry.ToLegacyEntry()

	if legacy == nil {
		t.Fatal("ToLegacyEntry returned nil")
	}

	if legacy.Metadata.Version != 5 {
		t.Errorf("Version mismatch: got %d, want 5", legacy.Metadata.Version)
	}
	if !legacy.Metadata.Created.Equal(created) {
		t.Errorf("Created mismatch")
	}
	if !legacy.Metadata.Updated.Equal(updated) {
		t.Errorf("Updated mismatch")
	}

	if legacy.Data["name"] != "Test Entry" {
		t.Errorf("name mismatch: got %v", legacy.Data["name"])
	}
	if legacy.Data["username"] != "testuser" {
		t.Errorf("username mismatch: got %v", legacy.Data["username"])
	}
	if legacy.Data["password"] != "secret123" {
		t.Errorf("password mismatch: got %v", legacy.Data["password"])
	}
	if legacy.Data["url"] != "https://example.com" {
		t.Errorf("url mismatch: got %v", legacy.Data["url"])
	}
	if legacy.Data["notes"] != "Test notes" {
		t.Errorf("notes mismatch: got %v", legacy.Data["notes"])
	}

	tags, ok := legacy.Data["tags"].([]string)
	if !ok {
		t.Fatalf("tags should be []string, got %T", legacy.Data["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("tags length mismatch: got %d, want 2", len(tags))
	}

	totpData, ok := legacy.Data["totp"].(map[string]any)
	if !ok {
		t.Fatalf("totp should be map[string]any, got %T", legacy.Data["totp"])
	}
	if totpData["secret"] != "JBSWY3DPEHPK3PXP" {
		t.Errorf("totp.secret mismatch")
	}

	fields, ok := legacy.Data["custom_fields"].(map[string]any)
	if !ok {
		t.Fatalf("custom_fields should be map[string]any, got %T", legacy.Data["custom_fields"])
	}
	if len(fields) != 2 {
		t.Errorf("custom_fields length mismatch: got %d, want 2", len(fields))
	}
}

func TestEntryV2ToLegacyEntryOmitsEmptyFields(t *testing.T) {
	entry := NewEntryV2()
	entry.Name = "Test"

	legacy := entry.ToLegacyEntry()

	if _, ok := legacy.Data["username"]; ok {
		t.Error("username should be omitted when empty")
	}
	if _, ok := legacy.Data["password"]; ok {
		t.Error("password should be omitted when empty")
	}
	if _, ok := legacy.Data["url"]; ok {
		t.Error("url should be omitted when empty")
	}
	if _, ok := legacy.Data["notes"]; ok {
		t.Error("notes should be omitted when empty")
	}
	if _, ok := legacy.Data["tags"]; ok {
		t.Error("tags should be omitted when empty")
	}
	if _, ok := legacy.Data["totp"]; ok {
		t.Error("totp should be omitted when nil")
	}
	if _, ok := legacy.Data["custom_fields"]; ok {
		t.Error("custom_fields should be omitted when empty")
	}
}

func TestEntryV2FromLegacy(t *testing.T) {
	created := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 3, 30, 10, 5, 0, 0, time.UTC)

	legacy := &Entry{
		Data: map[string]any{
			"name":     "Test Entry",
			"username": "testuser",
			"password": "secret123",
			"url":      "https://example.com",
			"notes":    "Test notes",
			"tags":     []any{"work", "test"},
			"totp": map[string]any{
				"secret":       "JBSWY3DPEHPK3PXP",
				"algorithm":    "SHA256",
				"digits":       float64(8),
				"period":       float64(60),
				"issuer":       "Example",
				"account_name": "testuser",
			},
			"custom_fields": map[string]any{
				"pin": map[string]any{
					"value": "1234",
					"type":  "hidden",
				},
			},
		},
		Metadata: EntryMetadata{
			Created: created,
			Updated: updated,
			Version: 5,
		},
	}

	entry := EntryV2FromLegacy(legacy)

	if entry == nil {
		t.Fatal("EntryV2FromLegacy returned nil")
	}

	if entry.Name != "Test Entry" {
		t.Errorf("Name mismatch: got %v, want %v", entry.Name, "Test Entry")
	}
	if entry.Username != "testuser" {
		t.Errorf("Username mismatch: got %v, want %v", entry.Username, "testuser")
	}
	if entry.Password != "secret123" {
		t.Errorf("Password mismatch: got %v, want %v", entry.Password, "secret123")
	}
	if entry.URL != "https://example.com" {
		t.Errorf("URL mismatch: got %v, want %v", entry.URL, "https://example.com")
	}
	if entry.Notes != "Test notes" {
		t.Errorf("Notes mismatch: got %v, want %v", entry.Notes, "Test notes")
	}
	if entry.Version != 5 {
		t.Errorf("Version mismatch: got %v, want %v", entry.Version, 5)
	}
	if !entry.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt mismatch")
	}
	if !entry.UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt mismatch")
	}

	if len(entry.Tags) != 2 {
		t.Errorf("Tags length mismatch: got %d, want 2", len(entry.Tags))
	}

	if entry.TOTP == nil {
		t.Fatal("TOTP should not be nil")
	}
	if entry.TOTP.Secret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("TOTP.Secret mismatch: got %v", entry.TOTP.Secret)
	}
	if entry.TOTP.Algorithm != "SHA256" {
		t.Errorf("TOTP.Algorithm mismatch: got %v", entry.TOTP.Algorithm)
	}
	if entry.TOTP.Digits != 8 {
		t.Errorf("TOTP.Digits mismatch: got %v, want 8", entry.TOTP.Digits)
	}
	if entry.TOTP.Period != 60 {
		t.Errorf("TOTP.Period mismatch: got %v, want 60", entry.TOTP.Period)
	}

	if len(entry.CustomFields) != 1 {
		t.Errorf("CustomFields length mismatch: got %d, want 1", len(entry.CustomFields))
	}
}

func TestEntryV2FromLegacyNilEntry(t *testing.T) {
	result := EntryV2FromLegacy(nil)
	if result != nil {
		t.Fatal("EntryV2FromLegacy(nil) should return nil")
	}
}

func TestEntryV2FromLegacyNilData(t *testing.T) {
	legacy := &Entry{
		Data: nil,
		Metadata: EntryMetadata{
			Created: time.Now(),
			Updated: time.Now(),
			Version: 1,
		},
	}

	entry := EntryV2FromLegacy(legacy)
	if entry == nil {
		t.Fatal("EntryV2FromLegacy should not return nil for nil data")
	}
	if len(entry.Tags) != 0 {
		t.Error("Tags should be empty")
	}
	if len(entry.CustomFields) != 0 {
		t.Error("CustomFields should be empty")
	}
}

func TestParseTOTPConfigDefaults(t *testing.T) {
	data := map[string]any{
		"secret": "JBSWY3DPEHPK3PXP",
	}

	totp := parseTOTPConfig(data)

	if totp.Secret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("Secret mismatch: got %v", totp.Secret)
	}
	if totp.Algorithm != "SHA1" {
		t.Errorf("Algorithm should default to SHA1, got %v", totp.Algorithm)
	}
	if totp.Digits != 6 {
		t.Errorf("Digits should default to 6, got %v", totp.Digits)
	}
	if totp.Period != 30 {
		t.Errorf("Period should default to 30, got %v", totp.Period)
	}
}

func TestParseTOTPConfigCustomValues(t *testing.T) {
	data := map[string]any{
		"secret":       "JBSWY3DPEHPK3PXP",
		"algorithm":    "SHA512",
		"digits":       float64(8),
		"period":       float64(60),
		"issuer":       "Custom",
		"account_name": "user@example.com",
	}

	totp := parseTOTPConfig(data)

	if totp.Algorithm != "SHA512" {
		t.Errorf("Algorithm mismatch: got %v", totp.Algorithm)
	}
	if totp.Digits != 8 {
		t.Errorf("Digits mismatch: got %v", totp.Digits)
	}
	if totp.Period != 60 {
		t.Errorf("Period mismatch: got %v", totp.Period)
	}
	if totp.Issuer != "Custom" {
		t.Errorf("Issuer mismatch: got %v", totp.Issuer)
	}
	if totp.AccountName != "user@example.com" {
		t.Errorf("AccountName mismatch: got %v", totp.AccountName)
	}
}

func TestIsStructuredEntry(t *testing.T) {
	tests := []struct {
		data     map[string]any
		name     string
		expected bool
	}{
		{
			name:     "nil data",
			data:     nil,
			expected: false,
		},
		{
			name:     "empty data",
			data:     map[string]any{},
			expected: false,
		},
		{
			name:     "only version",
			data:     map[string]any{"version": 1},
			expected: false,
		},
		{
			name:     "only created_at",
			data:     map[string]any{"created_at": "2026-01-01"},
			expected: false,
		},
		{
			name:     "both version and created_at",
			data:     map[string]any{"version": 1, "created_at": "2026-01-01"},
			expected: true,
		},
		{
			name:     "full structured data",
			data:     map[string]any{"version": 1, "created_at": "2026-01-01", "name": "test"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsStructuredEntry(tt.data)
			if got != tt.expected {
				t.Errorf("IsStructuredEntry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCustomFieldTypes(t *testing.T) {
	types := []CustomFieldType{
		FieldTypeString,
		FieldTypeHidden,
		FieldTypeURL,
		FieldTypeEmail,
		FieldTypeDate,
		FieldTypeNumber,
	}

	for _, ft := range types {
		field := CustomField{Type: ft}
		if field.Type != ft {
			t.Errorf("Field type mismatch for %v", ft)
		}
	}
}

func TestEntryV2ReadWriteRoundTrip(t *testing.T) {
	vaultDir := t.TempDir()
	id := testutil.TempIdentity(t)

	entry := &EntryV2{
		Name:     "GitHub",
		Username: "testuser",
		Password: "secretpassword",
		URL:      "https://github.com",
		Notes:    "My GitHub account",
		Tags:     []string{"work", "development"},
	}

	if err := WriteEntryV2(vaultDir, "github.com", entry, id); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	got, err := ReadEntryV2(vaultDir, "github.com", id)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}

	if got.Name != entry.Name {
		t.Errorf("Name mismatch: got %v, want %v", got.Name, entry.Name)
	}
	if got.Username != entry.Username {
		t.Errorf("Username mismatch: got %v, want %v", got.Username, entry.Username)
	}
	if got.Password != entry.Password {
		t.Errorf("Password mismatch: got %v, want %v", got.Password, entry.Password)
	}
	if got.URL != entry.URL {
		t.Errorf("URL mismatch: got %v, want %v", got.URL, entry.URL)
	}
	if got.Notes != entry.Notes {
		t.Errorf("Notes mismatch: got %v, want %v", got.Notes, entry.Notes)
	}
}

func TestEntryV2ReadWriteNilEntry(t *testing.T) {
	vaultDir := t.TempDir()
	id := testutil.TempIdentity(t)

	err := WriteEntryV2(vaultDir, "test", nil, id)
	if err == nil {
		t.Fatal("WriteEntryV2 should error on nil entry")
	}
}

func TestMergeEntryV2(t *testing.T) {
	vaultDir := t.TempDir()
	id := testutil.TempIdentity(t)

	entry := &EntryV2{
		Name:     "Test",
		Username: "original",
		Password: "oldpass",
	}
	if err := WriteEntryV2(vaultDir, "test", entry, id); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	merged, err := MergeEntryV2(vaultDir, "test", func(e *EntryV2) error {
		e.Password = "newpass"
		e.AddTag("updated")
		return nil
	}, id)

	if err != nil {
		t.Fatalf("merge entry: %v", err)
	}

	if merged.Username != "original" {
		t.Error("Username should be preserved")
	}
	if merged.Password != "newpass" {
		t.Error("Password should be updated")
	}
	if len(merged.Tags) != 1 || merged.Tags[0] != "updated" {
		t.Error("Tag should be added")
	}
	if merged.Version != 2 {
		t.Errorf("Version should be 2, got %d", merged.Version)
	}
}

func TestMergeEntryV2WithError(t *testing.T) {
	vaultDir := t.TempDir()
	id := testutil.TempIdentity(t)

	entry := &EntryV2{Name: "Test"}
	if err := WriteEntryV2(vaultDir, "test", entry, id); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	expectedErr := errors.New("merge function error")
	_, err := MergeEntryV2(vaultDir, "test", func(e *EntryV2) error {
		return expectedErr
	}, id)

	if err == nil {
		t.Fatal("MergeEntryV2 should propagate error")
	}
}

func TestReadEntryV2WithNilIdentity(t *testing.T) {
	vaultDir := t.TempDir()

	_, err := ReadEntryV2(vaultDir, "test", nil)
	if err == nil {
		t.Fatal("ReadEntryV2 should error on nil identity")
	}
}

func TestEntryGetField(t *testing.T) {
	entry := &Entry{
		Data: map[string]any{
			"username": "testuser",
			"password": "secret",
		},
	}

	val, ok := entry.GetField("username")
	if !ok {
		t.Fatal("should find username field")
	}
	if val != "testuser" {
		t.Errorf("username value mismatch: got %v", val)
	}

	_, ok = entry.GetField("nonexistent")
	if ok {
		t.Error("should not find nonexistent field")
	}

	entry.Data = nil
	_, ok = entry.GetField("username")
	if ok {
		t.Error("should not find field when data is nil")
	}
}

func TestEntrySetField(t *testing.T) {
	entry := &Entry{}

	entry.SetField("username", "testuser")
	if entry.Data == nil {
		t.Fatal("Data should be initialized")
	}
	if entry.Data["username"] != "testuser" {
		t.Errorf("username value mismatch")
	}

	entry.SetField("password", "secret")
	if entry.Data["password"] != "secret" {
		t.Errorf("password value mismatch")
	}
}

func TestEntryHasField(t *testing.T) {
	entry := &Entry{
		Data: map[string]any{
			"username": "testuser",
		},
	}

	if !entry.HasField("username") {
		t.Error("should have username field")
	}
	if entry.HasField("password") {
		t.Error("should not have password field")
	}

	entry.Data = nil
	if entry.HasField("username") {
		t.Error("should not have field when data is nil")
	}
}

func TestReadEntryWithNilIdentity(t *testing.T) {
	vaultDir := t.TempDir()

	_, err := ReadEntry(vaultDir, "test", nil)
	if err == nil {
		t.Fatal("ReadEntry should error on nil identity")
	}
}

func TestWriteEntryWithNilEntry(t *testing.T) {
	vaultDir := t.TempDir()
	id, _ := age.GenerateX25519Identity()

	err := WriteEntry(vaultDir, "test", nil, id)
	if err == nil {
		t.Fatal("WriteEntry should error on nil entry")
	}
}

func TestWriteEntryWithNilIdentity(t *testing.T) {
	vaultDir := t.TempDir()
	entry := &Entry{Data: map[string]any{"test": "value"}}

	err := WriteEntry(vaultDir, "test", entry, nil)
	if err == nil {
		t.Fatal("WriteEntry should error on nil identity")
	}
}

func TestEntryV2UnmarshalJSONError(t *testing.T) {
	data := []byte(`{invalid json}`)

	var entry EntryV2
	err := entry.UnmarshalJSON(data)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestEntryV2ToLegacyEntryWithTOTPError(t *testing.T) {
	entry := &EntryV2{
		Name: "Test",
		TOTP: &TOTPConfig{
			Secret: "JBSWY3DPEHPK3PXP",
		},
		Version: 1,
	}

	legacy := entry.ToLegacyEntry()
	if legacy == nil {
		t.Fatal("ToLegacyEntry should not return nil")
	}

	if legacy.Data["totp"] == nil {
		t.Error("totp should be present")
	}
}

func TestEntryV2ToLegacyEntryWithCustomFields(t *testing.T) {
	entry := &EntryV2{
		Name: "Test",
		CustomFields: []CustomField{
			{Name: "pin", Value: "1234", Type: FieldTypeHidden},
			{Name: "url", Value: "https://example.com", Type: FieldTypeURL},
		},
		Version: 1,
	}

	legacy := entry.ToLegacyEntry()
	if legacy == nil {
		t.Fatal("ToLegacyEntry should not return nil")
	}

	fields, ok := legacy.Data["custom_fields"].(map[string]any)
	if !ok {
		t.Fatal("custom_fields should be map[string]any")
	}
	if len(fields) != 2 {
		t.Errorf("custom_fields should have 2 entries, got %d", len(fields))
	}
}

func TestEntryV2FromLegacyWithPartialTOTP(t *testing.T) {
	legacy := &Entry{
		Data: map[string]any{
			"totp": map[string]any{
				"secret": "JBSWY3DPEHPK3PXP",
			},
		},
		Metadata: EntryMetadata{Version: 1},
	}

	entry := EntryV2FromLegacy(legacy)
	if entry == nil {
		t.Fatal("EntryV2FromLegacy should not return nil")
	}
	if entry.TOTP == nil {
		t.Fatal("TOTP should be parsed")
	}
	if entry.TOTP.Secret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("TOTP.Secret = %q, want %q", entry.TOTP.Secret, "JBSWY3DPEHPK3PXP")
	}
	if entry.TOTP.Algorithm != "SHA1" {
		t.Errorf("TOTP.Algorithm should default to SHA1, got %q", entry.TOTP.Algorithm)
	}
	if entry.TOTP.Digits != 6 {
		t.Errorf("TOTP.Digits should default to 6, got %d", entry.TOTP.Digits)
	}
}

func TestEntryV2FromLegacyWithInvalidTags(t *testing.T) {
	legacy := &Entry{
		Data: map[string]any{
			"tags": []any{"valid", 123, "also valid"},
		},
		Metadata: EntryMetadata{Version: 1},
	}

	entry := EntryV2FromLegacy(legacy)
	if entry == nil {
		t.Fatal("EntryV2FromLegacy should not return nil")
	}
	if len(entry.Tags) != 2 {
		t.Errorf("Tags should have 2 valid strings, got %d", len(entry.Tags))
	}
}

func TestMergeEntryV2WithNonexistentEntry(t *testing.T) {
	vaultDir := t.TempDir()
	id := testutil.TempIdentity(t)

	_, err := MergeEntryV2(vaultDir, "nonexistent", func(e *EntryV2) error {
		return nil
	}, id)
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
}

func TestMergeEntryV2MergeFnError(t *testing.T) {
	vaultDir := t.TempDir()
	id := testutil.TempIdentity(t)

	entry := &EntryV2{Name: "Test"}
	if err := WriteEntryV2(vaultDir, "test", entry, id); err != nil {
		t.Fatalf("WriteEntryV2 failed: %v", err)
	}

	expectedErr := errors.New("merge error")
	_, err := MergeEntryV2(vaultDir, "test", func(e *EntryV2) error {
		return expectedErr
	}, id)
	if err == nil {
		t.Fatal("expected error from merge function")
	}
}
