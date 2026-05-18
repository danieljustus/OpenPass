# Tool: autotype

Type a vault entry's field value as keyboard input into the currently focused application without exposing the value to the agent.

## USE WHEN
- The user needs to enter a credential into a login form or application
- You want to automate filling in credentials without the LLM seeing the value
- The user has the target application focused and ready

## DON'T USE WHEN
- The user needs to paste the value themselves → use copy_to_clipboard
- You need the value in the response → use get_entry_value
- No window is focused or autotype is not supported (headless) → use copy_to_clipboard
- You need to type multiple fields → call autotype once per field

## INPUT
- path (string, required): entry path (e.g. "github")
- field (string, optional, default "password"): field name to type (e.g. "password", "username")

## OUTPUT
```json
{
  "success": true,
  "path": "github",
  "field": "password"
}
```

## NOTES
- Requires `canUseAutotype: true` in agent profile
- The field value is never exposed in the MCP response
- Types into the currently focused application window
- Cross-platform: macOS, Linux (via xdotool), Windows (via AutoIt)

## COMBINES WELL WITH
- find_entries (discover path)
- get_entry_metadata (verify entry structure and field names)

## EXAMPLE
`{"path": "github", "field": "password"}` → password typed into focused app
