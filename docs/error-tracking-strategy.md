# Error Tracking Strategy for OpenPass

**Status**: DECIDED - No External Telemetry
**Last Updated**: 2026-04-19

## Decision

OpenPass does **NOT** use external error tracking services (Sentry, Datadog, Crashlytics, etc.) for the following reasons:

1. **Privacy-First**: As a password manager, OpenPass handles highly sensitive data. External telemetry services create unacceptable data exposure risks.
2. **Secret Safety**: Even with redaction, error reports from a密码 manager could inadvertently leak sensitive patterns.
3. **GDPR Compliance**: External telemetry would require explicit user consent and data processing agreements.
4. **User Trust**: Password manager users expect minimal network activity and no data exfiltration.

## Error Tracking Approach

### Local Error Bundles (Opt-In)

When users encounter errors, they can generate a local error bundle for self-diagnosis or sharing with maintainers:

```bash
openpass doctor --export-bundle /tmp/openpass-debug.tar.gz
```

The bundle contains:
- OpenPass version and Go version
- OS/platform information
- Redacted config (secrets stripped)
- Last 100 audit log entries (path patterns only, no field values)
- Stack trace (if available)
- Exit code and error category

### Redaction Rules

All error reports MUST redact the following before any export:

| Data Type | Redaction Rule |
|-----------|----------------|
| Field values | Replaced with `[REDACTED]` |
| Entry paths | Only parent directories exposed, entry names redacted |
| Field names | Normalized to generic names (e.g., `field.0`, `field.1`) |
| Header values | All headers redacted except content-type |
| Tokens/secrets | Always redacted, never exported |
| File contents | Never included in error reports |

### Error Categories and Exit Codes

| Code | Category | Description |
|------|----------|-------------|
| 0 | Success | Operation completed successfully |
| 1 | General Error | Unclassified error |
| 2 | Vault Error | Vault file corruption, missing identity, etc. |
| 3 | Crypto Error | Encryption/decryption failures, age errors |
| 4 | Config Error | Invalid configuration, missing config file |
| 5 | Keyring Error | OS keyring access failures |
| 6 | Git Error | Git operation failures |
| 7 | Network Error | HTTP/MCP network issues |
| 8 | Permission Error | File/directory permission issues |
| 9 | Validation Error | Input validation failures |
| 10 | MCP Error | MCP server or protocol errors |
| 11 | Audit Error | Audit logging failures |

### CLI Error Output Format

```
Error: failed to decrypt entry
Category: crypto_error (exit code 3)
Suggestion: This may indicate vault corruption. Try 'openpass doctor' for diagnostics.
```

### MCP Error Responses

MCP errors include sanitized error data:

```json
{
  "error": {
    "code": "CRYPTO_ERROR",
    "message": "decryption failed",
    "category": 3,
    "suggestion": "Verify vault integrity with 'openpass doctor'"
  }
}
```

## Implementation Status

- [x] Error category enum defined
- [x] Exit codes documented
- [x] MCP error response format defined
- [ ] Local error bundle command (`openpass doctor --export-bundle`)
- [ ] Redaction utilities for error reports
- [ ] Audit log export with redaction