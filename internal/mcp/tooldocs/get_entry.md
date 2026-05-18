# Tool: get_entry

Get metadata for a vault entry. Returns type, usage hints, and field names without secret values. Use `get_entry_value` to retrieve actual values.

## USE WHEN
- You need to see field names and metadata for an entry
- You want to check entry type, tags, and usage hints
- You know the exact path (use find_entries first if unsure)
- You need to decide which field to request via get_entry_value

## DON'T USE WHEN
- You need the actual secret values → use get_entry_value
- You only need version/update info → use get_entry_metadata (lighter)
- You want to search by keyword → use find_entries first
- You need to use the value without seeing it → use copy_to_clipboard or autotype

## INPUT
- path (string, required): entry path (e.g. "github", "work/aws")
- include_value (boolean, optional, default false): when true, returns the full entry with secret values (requires higher approval)

## OUTPUT
```json
{
  "path": "github",
  "type": "password",
  "usage_hint": "Personal GitHub account",
  "auto_rotate": false,
  "fields": [{"name": "password", "handle": "op://github/password", "kind": "string"}, ...],
  "has_value": true,
  "tags": ["web"],
  "meta": {"created": "...", "updated": "...", "version": 3}
}
```

## COMBINES WELL WITH
- find_entries (discover path first)
- get_entry_value (read specific field after inspecting structure)
- get_entry_metadata (lighter alternative when only version/timestamps needed)
- copy_to_clipboard (consume password without exposing to LLM)

## EXAMPLE
`{"path": "github"}` → entry metadata with field names and version info
