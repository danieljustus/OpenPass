# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.0.0] - 2026-04-22

Initial stable OpenPass release.

### Added

- Age-based encrypted vault storage with passphrase-protected identity files
- CLI commands for initializing vaults, adding, setting, getting, listing, finding, editing, deleting, and generating entries
- TOTP support for storing seeds and generating one-time codes
- Clipboard copy support with automatic clearing
- Session caching through the operating system keyring
- Git integration for vault history and synchronization
- Multi-recipient vault support for shared access
- MCP server support over stdio and local HTTP transports
- Agent profiles, path restrictions, write controls, metadata-only reads, and TOTP-safe redaction
- HTTP MCP bearer token authentication, request validation, health checks, and Prometheus metrics
- JSON output for automation-friendly CLI use
- Shell completions and generated manual pages
- Linux package, archive, and checksum release automation
- CI coverage checks, race tests, smoke tests, vulnerability scanning, and linting

### Security

- Entry files are encrypted independently with age X25519 and ChaCha20-Poly1305
- Passphrases are read through terminal password prompts and are not stored in plain text
- HTTP MCP binds to `127.0.0.1` by default and requires bearer token authentication
- Release checksums are published for artifact verification

[v1.0.0]: https://github.com/danieljustus/OpenPass/releases/tag/v1.0.0
