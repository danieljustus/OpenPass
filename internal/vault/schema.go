package vault

import (
	"encoding/json"
	"time"
)

// EntryV2 represents a structured password entry with typed fields.
// This is the new schema that provides better type safety and structure
// compared to the legacy Entry type which used map[string]any.
type EntryV2 struct {
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	TOTP         *TOTPConfig   `json:"totp,omitempty"`
	Name         string        `json:"name,omitempty"`
	Username     string        `json:"username,omitempty"`
	Password     string        `json:"password,omitempty"`
	URL          string        `json:"url,omitempty"`
	Notes        string        `json:"notes,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
	CustomFields []CustomField `json:"custom_fields,omitempty"`
	Version      int           `json:"version"`
}

// TOTPConfig represents configuration for time-based one-time passwords
type TOTPConfig struct {
	Secret      string `json:"secret"`
	Algorithm   string `json:"algorithm,omitempty"`
	Issuer      string `json:"issuer,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	Digits      int    `json:"digits,omitempty"`
	Period      int    `json:"period,omitempty"`
}

// CustomFieldType represents the type of a custom field
type CustomFieldType string

const (
	// FieldTypeString is a plain text field
	FieldTypeString CustomFieldType = "string"

	// FieldTypeHidden is a concealed field (like a second password)
	FieldTypeHidden CustomFieldType = "hidden"

	// FieldTypeURL is a URL field
	FieldTypeURL CustomFieldType = "url"

	// FieldTypeEmail is an email address field
	FieldTypeEmail CustomFieldType = "email"

	// FieldTypeDate is a date field
	FieldTypeDate CustomFieldType = "date"

	// FieldTypeNumber is a numeric field
	FieldTypeNumber CustomFieldType = "number"
)

// CustomField represents a user-defined field with a type
type CustomField struct {
	// Name is the field identifier
	Name string `json:"name"`

	// Value is the field content
	Value string `json:"value"`

	// Type indicates how the field should be displayed/handled
	Type CustomFieldType `json:"type,omitempty"`
}

// NewEntryV2 creates a new EntryV2 with initialized timestamps
func NewEntryV2() *EntryV2 {
	now := time.Now().UTC()
	return &EntryV2{
		Tags:         []string{},
		CustomFields: []CustomField{},
		CreatedAt:    now,
		UpdatedAt:    now,
		Version:      1,
	}
}

// UpdateTimestamps updates the UpdatedAt timestamp and increments version
func (e *EntryV2) UpdateTimestamps() {
	e.UpdatedAt = time.Now().UTC()
	e.Version++
}

// AddTag adds a tag if it doesn't already exist
func (e *EntryV2) AddTag(tag string) {
	for _, t := range e.Tags {
		if t == tag {
			return
		}
	}
	e.Tags = append(e.Tags, tag)
}

// RemoveTag removes a tag if it exists
func (e *EntryV2) RemoveTag(tag string) {
	filtered := make([]string, 0, len(e.Tags))
	for _, t := range e.Tags {
		if t != tag {
			filtered = append(filtered, t)
		}
	}
	e.Tags = filtered
}

// AddCustomField adds or updates a custom field
func (e *EntryV2) AddCustomField(field CustomField) {
	for i, f := range e.CustomFields {
		if f.Name == field.Name {
			e.CustomFields[i] = field
			return
		}
	}
	e.CustomFields = append(e.CustomFields, field)
}

// GetCustomField retrieves a custom field by name
func (e *EntryV2) GetCustomField(name string) (CustomField, bool) {
	for _, f := range e.CustomFields {
		if f.Name == name {
			return f, true
		}
	}
	return CustomField{}, false
}

// RemoveCustomField removes a custom field by name
func (e *EntryV2) RemoveCustomField(name string) {
	filtered := make([]CustomField, 0, len(e.CustomFields))
	for _, f := range e.CustomFields {
		if f.Name != name {
			filtered = append(filtered, f)
		}
	}
	e.CustomFields = filtered
}

// MarshalJSON implements custom JSON marshaling for EntryV2
func (e EntryV2) MarshalJSON() ([]byte, error) {
	type alias EntryV2
	return json.Marshal(alias(e))
}

// UnmarshalJSON implements custom JSON unmarshaling for EntryV2
func (e *EntryV2) UnmarshalJSON(data []byte) error {
	type alias EntryV2
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*e = EntryV2(v)

	// Ensure slices are initialized
	if e.Tags == nil {
		e.Tags = []string{}
	}
	if e.CustomFields == nil {
		e.CustomFields = []CustomField{}
	}

	return nil
}

// ToLegacyEntry converts an EntryV2 to the legacy Entry format
// This maintains backward compatibility with existing vault data
func (e *EntryV2) ToLegacyEntry() *Entry {
	data := make(map[string]any)

	if e.Name != "" {
		data["name"] = e.Name
	}
	if e.Username != "" {
		data["username"] = e.Username
	}
	if e.Password != "" {
		data["password"] = e.Password
	}
	if e.URL != "" {
		data["url"] = e.URL
	}
	if e.Notes != "" {
		data["notes"] = e.Notes
	}
	if len(e.Tags) > 0 {
		data["tags"] = e.Tags
	}
	if e.TOTP != nil {
		totpData, err := json.Marshal(e.TOTP) //nosec:G117
		if err == nil {
			var totpMap map[string]any
			if err := json.Unmarshal(totpData, &totpMap); err != nil {
				totpMap = nil
			}
			data["totp"] = totpMap
		}
	}
	if len(e.CustomFields) > 0 {
		fields := make(map[string]any)
		for _, f := range e.CustomFields {
			fields[f.Name] = map[string]any{
				"value": f.Value,
				"type":  string(f.Type),
			}
		}
		data["custom_fields"] = fields
	}

	return &Entry{
		Data: data,
		Metadata: EntryMetadata{
			Created: e.CreatedAt,
			Updated: e.UpdatedAt,
			Version: e.Version,
		},
	}
}

// EntryV2FromLegacy creates an EntryV2 from a legacy Entry
// This allows migration from the old format to the new structured format
func EntryV2FromLegacy(entry *Entry) *EntryV2 {
	if entry == nil {
		return nil
	}

	e := &EntryV2{
		Tags:         []string{},
		CustomFields: []CustomField{},
		CreatedAt:    entry.Metadata.Created,
		UpdatedAt:    entry.Metadata.Updated,
		Version:      entry.Metadata.Version,
	}

	if entry.Data == nil {
		return e
	}

	// Extract known fields
	if v, ok := entry.Data["name"].(string); ok {
		e.Name = v
	}
	if v, ok := entry.Data["username"].(string); ok {
		e.Username = v
	}
	if v, ok := entry.Data["password"].(string); ok {
		e.Password = v
	}
	if v, ok := entry.Data["url"].(string); ok {
		e.URL = v
	}
	if v, ok := entry.Data["notes"].(string); ok {
		e.Notes = v
	}

	// Extract tags
	if tags, ok := entry.Data["tags"].([]any); ok {
		for _, t := range tags {
			if tag, ok := t.(string); ok {
				e.Tags = append(e.Tags, tag)
			}
		}
	}

	// Extract TOTP config
	if totpData, ok := entry.Data["totp"].(map[string]any); ok {
		e.TOTP = parseTOTPConfig(totpData)
	}

	// Extract custom fields
	if fields, ok := entry.Data["custom_fields"].(map[string]any); ok {
		for name, fieldData := range fields {
			if fieldMap, ok := fieldData.(map[string]any); ok {
				cf := CustomField{Name: name}
				if v, ok := fieldMap["value"].(string); ok {
					cf.Value = v
				}
				if v, ok := fieldMap["type"].(string); ok {
					cf.Type = CustomFieldType(v)
				}
				e.CustomFields = append(e.CustomFields, cf)
			}
		}
	}

	return e
}

// parseTOTPConfig parses TOTP configuration from map data
func parseTOTPConfig(data map[string]any) *TOTPConfig {
	totp := &TOTPConfig{}

	if v, ok := data["secret"].(string); ok {
		totp.Secret = v
	}
	if v, ok := data["algorithm"].(string); ok {
		totp.Algorithm = v
	}
	if v, ok := data["issuer"].(string); ok {
		totp.Issuer = v
	}
	if v, ok := data["account_name"].(string); ok {
		totp.AccountName = v
	}
	if v, ok := data["digits"].(float64); ok {
		totp.Digits = int(v)
	}
	if v, ok := data["period"].(float64); ok {
		totp.Period = int(v)
	}

	// Set defaults
	if totp.Algorithm == "" {
		totp.Algorithm = "SHA1"
	}
	if totp.Digits == 0 {
		totp.Digits = 6
	}
	if totp.Period == 0 {
		totp.Period = 30
	}

	return totp
}

// IsStructuredEntry checks if the given data represents a structured EntryV2
// by looking for the presence of version field and created_at timestamp
func IsStructuredEntry(data map[string]any) bool {
	if data == nil {
		return false
	}
	// Check for version field (always present in EntryV2)
	if _, hasVersion := data["version"]; !hasVersion {
		return false
	}
	// Check for created_at field
	if _, hasCreated := data["created_at"]; !hasCreated {
		return false
	}
	return true
}
