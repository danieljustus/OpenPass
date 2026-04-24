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

## [v1.1.0] - 2026-04-23

Major update with vault improvements, self-update mechanism, and enhanced MCP transport.

### Added

- Update check command (`openpass update check`) for detecting newer releases
- Self-update mechanism for managing OpenPass installations
- MCP server stdio transport support for local agent integration
- Session management commands (`openpass unlock`, `openpass lock`) with configurable TTL
- Release smoke tests for validating published artifacts
- Installer scripts for cross-platform installation (`install.sh`, `install.ps1`)

### Changed

- Vault structure refactored to use `entries/` subdirectory for organized storage
- Entry format updated to structured YAML with individual file encryption
- Removed index cache in favor of direct filesystem operations
- Improved handler concurrency for better performance
- Enhanced context propagation throughout the codebase

### Fixed

- Audit error logging improved for better diagnostics

## [v1.1.1] - 2026-04-24

Documentation updates.

### Changed

- Updated documentation images and README content

## [v1.1.2] - 2026-04-24

Documentation fixes.

### Changed

- Additional documentation images and README updates (same commit as v1.1.1)

## [v1.1.3] - 2026-04-24

CI configuration fix.

### Fixed

- Skipped Homebrew tap publishing workflow until GitHub PAT is properly configured

## [v1.1.4] - 2026-04-24

CI fix for release validation.

### Fixed

- Fixed release smoke tests to properly validate published artifacts

[v1.0.0]: https://github.com/danieljustus/OpenPass/releases/tag/v1.0.0
[v1.1.0]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.0
[v1.1.1]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.1
[v1.1.2]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.2
[v1.1.3]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.3
[v1.1.4]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.4
