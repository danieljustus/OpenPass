# Configuration Reference

Global config is stored at `~/.openpass/config.yaml`. Vault-specific config is stored in the vault directory.
Use [`config.yaml.example`](../config.yaml.example) as a commented starting point.

## Environment Variables

- `OPENPASS_VAULT` — Path to vault directory (default: `~/.openpass`)

Or use the `--vault` flag to override for any command:
```bash
openpass --vault ~/work-vault get aws.secret
```

## config.yaml

```yaml
# ~/.openpass/config.yaml — Global configuration

# Default vault directory
vaultDir: ~/.openpass

# Default agent for MCP (can be overridden via --agent flag)
defaultAgent: default

# Session timeout for OS keyring cache (default: 15m)
sessionTimeout: 15m

# Agent profiles for MCP server
agents:
  default:
    allowedPaths: ["*"]
    canWrite: false
    approvalMode: none
  claude-code:
    allowedPaths: ["*"]
    canWrite: true
    approvalMode: none

# Vault-specific configuration (optional, can also be in vault/config.yaml)
vault:
  path: ~/my-vault
  default_recipients:
    - age1...

# Git configuration
git:
  auto_push: true
  commit_template: "Update from OpenPass"

# MCP server configuration
mcp:
  port: 8080
  bind: 127.0.0.1
  stdio: false
  httpTokenFile: auto
```

## Config Options

| Option | Default | Description |
|--------|---------|-------------|
| `vaultDir` | `~/.openpass` | Default vault directory |
| `defaultAgent` | `default` | Default MCP agent profile |
| `sessionTimeout` | `15m` | OS keyring cache TTL |

## Agent Profile Options

| Option | Description |
|--------|-------------|
| `allowedPaths` | Path patterns the agent can access (prefix patterns, `*` for all) |
| `canWrite` | Whether the agent can create/update/delete entries |
| `approvalMode` | `none` (allow all), `deny` (reject writes), `prompt` (degrades to deny in MCP) |

## Vault Config Options

| Option | Description |
|--------|-------------|
| `path` | Vault directory path |
| `default_recipients` | Default age recipients for new entries |
| `confirm_remove` | Ask for confirmation before removing recipients |

## Git Config Options

| Option | Default | Description |
|--------|---------|-------------|
| `auto_push` | `true` | Automatically push after commit |
| `commit_template` | `"Update from OpenPass"` | Commit message template |

## MCP Config Options

| Option | Default | Description |
|--------|---------|-------------|
| `port` | `8080` | HTTP server port |
| `bind` | `127.0.0.1` | Bind address |
| `stdio` | `false` | Enable stdio transport |
| `httpTokenFile` | `auto` | Bearer token file path |

## Clipboard Config Options

| Option | Default | Description |
|--------|---------|-------------|
| `auto_clear_duration` | `30` | Seconds before copied secrets are cleared; `0` disables auto-clear |
