# Tool: copy_to_clipboard

Copy a vault entry's password field to the system clipboard without exposing the value to the agent.

## USE WHEN
- The user needs to paste a password somewhere
- You want to deliver a secret without it appearing in the conversation
- You've identified the correct entry path via find_entries or get_entry_metadata

## DON'T USE WHEN
- You need to type the password into a focused app → use autotype
- You need the actual value in the response → use get_entry_value
- You just need to check if an entry exists → use get_entry_metadata
- Clipboard is not available (headless server) → use get_entry_value as fallback

## INPUT
- path (string, required): entry path whose password field to copy

## OUTPUT
```json
{
  "success": true,
  "path": "github",
  "clears_at": "2026-05-18T10:30:30Z"
}
```

## NOTES
- Requires `canUseClipboard: true` in agent profile
- The password value is never exposed in the MCP response
- Clipboard is automatically cleared after 30 seconds
- Only copies the `password` field of the entry

## COMBINES WELL WITH
- find_entries (discover path)
- get_entry_metadata (verify entry before clipboard)

## EXAMPLE
`{"path": "github"}` → password copied to clipboard, confirmation returned
