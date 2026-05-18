# Tool: get_entry_metadata

Get metadata for a vault entry without retrieving sensitive data.

## USE WHEN
- You need to confirm an entry exists at a given path
- You want to check the version number for cache validation
- You need field names and types without reading values
- You want to see entry tags, type, and usage hints

## DON'T USE WHEN
- You need the actual secret values → use get_entry or get_entry_value
- You don't know the exact path → use find_entries first
- You only need the entry list → use list_entries

## INPUT
- path (string, required): entry path (e.g. "github", "work/aws")

## OUTPUT
```json
{
  "path": "github",
  "type": "password",
  "usage_hint": "<!-- DATA_abc123 label=usage_hint -->Personal GitHub account<!-- /DATA_abc123 -->",
  "auto_rotate": false,
  "fields": [
    {"name": "password", "handle": "op://github/password", "kind": "string"},
    {"name": "username", "handle": "op://github/username", "kind": "string"}
  ],
  "has_value": true,
  "tags": ["<!-- DATA_def456 label=tag -->web<!-- /DATA_def456 -->"],
  "meta": {
    "created": "2026-01-15T14:32:00Z",
    "updated": "2026-05-01T12:00:00Z",
    "version": 3
  }
}
```

## COMBINES WELL WITH
- find_entries (discover path first)
- get_entry (read after verifying metadata)
- copy_to_clipboard (skip value exposure if metadata confirms the entry)

## EXAMPLE
`{"path": "github"}` → metadata with version, fields, tags
