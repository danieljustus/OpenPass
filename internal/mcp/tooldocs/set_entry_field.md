# Tool: set_entry_field

Set a field on an entry. Requires write scope.

## USE WHEN
- You need to create or update a credential field
- The user asks you to store a new password, API key, or other secret
- You need to update an existing entry with new information

## DON'T USE WHEN
- The value is sensitive and the user should type it themselves → use secure_input or request_credential
- You only want to read → use get_entry, get_entry_metadata, or find_entries
- You need to delete a field or entry → use delete_entry

## INPUT
- path (string, required): entry path (e.g. "github", "work/aws")
- field (string, required): field name (e.g. "password", "username", "api_key")
- value (string, required): field value
- force (boolean, optional, default false): skip password strength validation

## OUTPUT
```json
{
  "success": true,
  "path": "github",
  "field": "password",
  "version": 6
}
```

## NOTES
- Creates entry if it doesn't exist
- Updates existing field or adds new one
- Triggers automatic git commit (if enabled)
- Requires `canWrite: true` in agent profile
- Value strength is validated unless `force: true`

## COMBINES WELL WITH
- find_entries (check if entry exists before updating)
- generate_password (generate a strong password to store)
- secure_input or request_credential (alternative for user-provided values)

## EXAMPLE
`{"path": "github", "field": "password", "value": "my-new-password"}` → field updated
