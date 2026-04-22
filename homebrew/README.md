# OpenPass Homebrew Formula

Local Homebrew formula support for [OpenPass](https://github.com/danieljustus/OpenPass) - a modern CLI password manager built with Go and age encryption.

## Installation

Use the release binaries from GitHub or install from source with Go. The formula
in this directory is intended for local formula validation and future tap
publishing.

## Testing the v1.0.0 formula locally

The checked-in formula builds from the `v1.0.0` source tag and is intended for local formula testing.

Homebrew 5 requires formula files to live in a tap. From the repository root, create a throwaway local tap and install from it:

```bash
brew tap-new local/openpass || true
cp homebrew/Formula/openpass.rb "$(brew --repository)/Library/Taps/local/homebrew-openpass/Formula/openpass.rb"
brew install --build-from-source local/openpass/openpass
brew test local/openpass/openpass
openpass version
```

To reinstall after editing the formula:

```bash
brew uninstall openpass
cp homebrew/Formula/openpass.rb "$(brew --repository)/Library/Taps/local/homebrew-openpass/Formula/openpass.rb"
brew install --build-from-source local/openpass/openpass
brew test local/openpass/openpass
```

To remove the local test install:

```bash
brew uninstall openpass
brew untap local/openpass
```

## Features

- 🔐 **Age encryption** - Modern, simple, and secure file encryption
- 🖥️ **CLI-first** - Fast, scriptable, and works everywhere
- 🔑 **MCP server** - AI agent integration via Model Context Protocol
- 👥 **Multi-user** - Share vaults with team members
- 🔄 **Git integration** - Version control your passwords
- 🛡️ **TOTP support** - Generate 2FA codes
- 📋 **Clipboard** - Auto-clear for secure copying

## Quick Start

```bash
# Initialize a new vault
openpass init

# Add your first entry
openpass set github.com/username

# Retrieve it
openpass get github.com/username

# Generate a password
openpass generate

# Set up MCP server for AI agents
openpass mcp-config claude-code
```

## Documentation

Full documentation is available at: https://github.com/danieljustus/OpenPass

## License

MIT License - see https://github.com/danieljustus/OpenPass/blob/main/LICENSE
