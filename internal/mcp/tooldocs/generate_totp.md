# Tool: generate_totp

Generate a TOTP code for an entry with TOTP configuration.

## USE WHEN
- You need a current TOTP/2FA code for authentication
- The entry is known to have TOTP configured (has a totp.secret field)
- The user needs to log in somewhere that requires 2FA

## DON'T USE WHEN
- You need the raw TOTP secret (almost never needed) → use get_entry_value
- You need to set up TOTP on an entry → use set_entry_field
- The entry doesn't have TOTP configured → check with get_entry_metadata first

## INPUT
- path (string, required): entry path with TOTP configuration
- destination (string, optional, "clipboard"|"autotype"|"return", default "clipboard"): where to send the code
- return_code (boolean, optional): must be true when destination="return"

## OUTPUT (destination=clipboard, default)
```json
{
  "success": true,
  "path": "github",
  "destination": "clipboard"
}
```

## OUTPUT (destination=return)
```json
{
  "code": "123456",
  "expires_at": "2026-05-18T10:30:30Z",
  "period": 30
}
```

## NOTES
- By default the code is copied to the clipboard without being returned in the response
- Use `destination="autotype"` to type the code directly into the focused app
- Use `destination="return"` with `return_code=true` to get the code in the response (requires approval)
- This tool only returns the generated code, not the underlying TOTP secret
- Works even when `redactFields` hides the TOTP secret from get_entry

## COMBINES WELL WITH
- get_entry_metadata (confirm TOTP is configured on the entry)
- autotype (type the code into a login form)

## EXAMPLE
`{"path": "github"}` → code copied to clipboard
`{"path": "github", "destination": "return", "return_code": true}` → `{"code": "123456", ...}`
