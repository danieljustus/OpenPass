# Tool: get_entry_value

Get the actual secret values for a vault entry. Use with caution — only request values when absolutely needed.

## USE WHEN
- You need the actual secret value (password, API key, token) to use it
- You've already inspected the entry structure via get_entry or get_entry_metadata
- The user explicitly asked you to retrieve a specific credential

## DON'T USE WHEN
- You only need metadata or field names → use get_entry or get_entry_metadata
- You can use the value without seeing it → use copy_to_clipboard or autotype
- You're just checking if an entry exists → use get_entry_metadata
- The value can be passed to a command without exposing it → use run_command or execute_with_secret

## INPUT
- path (string, required): entry path (e.g. "github", "work/aws")

## OUTPUT
Full entry data including secret values, wrapped with injection-safe markers.

## RISK LEVEL
**High** — This tool exposes raw secret values. Many tier presets restrict or deny this tool. Values are embedded with tamper-evident markers to prevent prompt injection.

## COMBINES WELL WITH
- get_entry or get_entry_metadata (verify path and field names first)
- run_command or execute_with_secret (alternative that avoids exposing values)

## EXAMPLE
`{"path": "github"}` → full entry data with all field values
