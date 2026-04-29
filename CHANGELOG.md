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

## [v1.2.0] - 2026-04-24

Audit, backup, and security hardening release.

### Added

- Backup and restore commands with automated test coverage
- Audit log support and broader integration test coverage for vault operations

### Changed

- Enabled Homebrew tap publishing through the release workflow
- Raised package test coverage across vault, session, update, git, and audit paths

### Fixed

- Resolved CI lint failures and a serve race condition
- Fixed generated coverage artifact handling in Git ignores

### Security

- Hardened file handling against path traversal, symlink TOCTOU, and unsafe permissions
- Addressed gosec findings for integer conversion and weak crypto hash usage

## [v1.3.0] - 2026-04-26

MCP, vault search, and backup hardening release.

### Added

- Concurrent vault search with scoped `FindWithOptions` support
- Trusted proxy support for MCP HTTP deployments
- TOTP secret validation before storing credentials

### Changed

- Metrics endpoint authentication now respects loopback vs non-loopback bind security

### Fixed

- Password generation stores entries through the vault entry path helper

### Security

- Hardened backup restore against symlink and permission vulnerabilities
- Fixed additional integer overflow findings in backup restore handling

## [v2.0.0] - 2026-04-28

Platform expansion, transport hardening, and developer experience release.

### Added

- FreeBSD support with in-memory encrypted session cache fallback (AES-256-GCM)
- Scoop distribution channel for Windows
- TLS configuration support for update checker and HTTP MCP
- Batch MCP operations for efficient multi-entry workflows
- Quiet mode flag (`-q` / `--quiet`) for script-friendly output
- Sentinel errors and restructured exit codes for reliable automation
- Password normalization and enhanced TOTP metadata support
- Improved in-memory session cache with configurable expiration
- Graceful OS keyring fallback to memory cache when keyring is unavailable

### Changed

- Go version bumped to 1.26.2
- Comprehensive CI documentation with lint gates and FreeBSD guidance

### Fixed

- FreeBSD cross-compilation by isolating go-keyring imports
- golangci-lint findings across bodyclose, errorlint, shadow, and staticcheck
- Session prompt logic now checks active session before requesting passphrase

## [v2.1.0] - 2026-04-29

Interactive TUI, vault management, and observability release.

### Added

- Interactive TUI using bubbletea for intuitive vault browsing and management
- Import command for migrating from 1Password, Bitwarden, pass, and CSV
- Config command with JSON schema validation for structured settings
- Profile command for multi-vault management and switching
- Auth command for passphrase and biometric (Touch ID) authentication setup
- Vault service layer for cleaner separation of concerns
- OpenTelemetry tracing support for operation observability
- Structured logging package with configurable verbosity
- Diag command for runtime metrics and environment inspection
- Windows path sanitization in importer to strip invalid characters

### Changed

- Refactored MCP server into focused files for maintainability
- Moved clipboard package to `internal/clipboard` for better organization
- Updated project documentation and generated man pages

### Fixed

- Resolved golangci-lint v2 compatibility issues
- Fixed Ubuntu CI session caching test flakiness
- Corrected release workflow environment variable duplication

[v1.0.0]: https://github.com/danieljustus/OpenPass/releases/tag/v1.0.0
[v1.1.0]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.0
[v1.1.1]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.1
[v1.1.2]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.2
[v1.1.3]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.3
[v1.1.4]: https://github.com/danieljustus/OpenPass/releases/tag/v1.1.4
[v1.2.0]: https://github.com/danieljustus/OpenPass/releases/tag/v1.2.0
[v1.3.0]: https://github.com/danieljustus/OpenPass/releases/tag/v1.3.0
[v2.0.0]: https://github.com/danieljustus/OpenPass/releases/tag/v2.0.0
[v2.1.0]: https://github.com/danieljustus/OpenPass/releases/tag/v2.1.0
