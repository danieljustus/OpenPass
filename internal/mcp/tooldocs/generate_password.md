# Tool: generate_password

Generate a secure password.

## USE WHEN
- The user needs a new strong password
- You're creating a new account and need to generate credentials
- You need a random token or secret for configuration

## DON'T USE WHEN
- You already have a password to store → use set_entry_field
- The user needs a TOTP code → use generate_totp
- You need a password stored in the vault → generate then store with set_entry_field

## INPUT
- length (number, optional, default 16): password length (8-128)
- symbols (boolean, optional, default true): include special characters

## OUTPUT
```json
{
  "password": "xK9#mP2$vL7@nQ4!aB8&"
}
```

## COMBINES WELL WITH
- set_entry_field (store the generated password)
- secure_input or request_credential (if user wants to provide their own)

## EXAMPLE
`{"length": 32, "symbols": true}` → `{"password": "aB3#xK9!mP2$vL7@nQ4&wR5%tY8*cF1?eD0"}`

## EXAMPLE
`{"length": 20, "symbols": false}` → alphanumeric password (no symbols)
