# OpenPass Troubleshooting Guide

This guide covers common issues you may encounter when using OpenPass and provides diagnostic steps and solutions.

## Table of Contents

1. [Quick Fixes](#quick-fixes)
2. [Vault Access Issues](#vault-access-issues)
3. [MCP Connection Problems](#mcp-connection-problems)
4. [Git Sync Issues](#git-sync-issues)
5. [Platform-Specific Issues](#platform-specific-issues)
6. [Performance Issues](#performance-issues)
7. [Diagnostic Commands Reference](#diagnostic-commands-reference)
8. [Before You Open an Issue](#before-you-open-an-issue)

---

## Quick Fixes

Before diving into detailed diagnostics, try these common solutions:

| Issue | Quick Fix |
|-------|-----------|
| Agent can't connect | Restart the MCP server: `openpass serve --stdio --agent default` |
| Permission denied | Check agent profile in `~/.openpass/config.yaml` |
| Vault locked | Run `openpass unlock` and enter your passphrase |
| Slow response | Check if vault has too many entries; consider organizing into subdirectories |
| Token invalid | Regenerate with `openpass mcp-config <agent> --http` |
| Changes not syncing | Run `openpass git push` manually |

**General restart sequence:**
```bash
openpass lock          # Clear cached passphrase
openpass unlock        # Re-authenticate
openpass serve --stdio --agent default  # Restart MCP server
```

---

## Vault Access Issues

### Symptom: "Vault locked" or "Failed to decrypt identity"

**Diagnostic steps:**

1. Check if vault is initialized:
   ```bash
   ls -la ~/.openpass/
   ```
   You should see `identity.age`, `config.yaml`, and `entries/` directory.

2. Verify identity file exists and is readable:
   ```bash
   ls -la ~/.openpass/identity.age
   file ~/.openpass/identity.age
   ```

3. Test passphrase entry:
   ```bash
   openpass unlock
   ```

**Solutions:**

| Problem | Solution |
|---------|----------|
| Vault not initialized | Run `openpass init` to create a new vault |
| Wrong passphrase | Try again carefully; if forgotten, restore from backup |
| Corrupted `identity.age` | Restore from backup; without backup, vault is unrecoverable |
| Missing `identity.age` | Check if vault path is correct: `openpass --vault /path get test` |
| Permission denied on identity file | Fix permissions: `chmod 600 ~/.openpass/identity.age` |

**Session caching issues:**
- If `openpass unlock` works but MCP server still reports locked, the session cache may have expired
- Default TTL is 15 minutes; extend with: `openpass unlock --ttl 30m`
- Clear cache and retry: `openpass lock && openpass unlock`

---

## MCP Connection Problems

### Symptom: Agent can't connect to OpenPass

**Diagnostic steps:**

1. **Verify MCP server is running:**
   ```bash
   # For stdio mode
   openpass serve --stdio --agent default
   
   # For HTTP mode
   openpass serve --port 8080
   ```

2. **Check HTTP server health:**
   ```bash
   curl -s http://127.0.0.1:8080/health
   ```

3. **Verify token file exists (HTTP mode):**
   ```bash
   ls -la ~/.openpass/mcp-token
   cat ~/.openpass/mcp-token
   ```

4. **Check agent profile configuration:**
   ```bash
   cat ~/.openpass/config.yaml | grep -A 5 "agents:"
   ```

5. **Verify agent name matches:**
   - The `--agent` flag must match a profile in `config.yaml`
   - For HTTP mode, the `X-OpenPass-Agent` header must match a profile

**Common MCP issues and solutions:**

| Problem | Solution |
|---------|----------|
| "Agent not recognized" | Verify agent name in `--agent` flag matches `config.yaml` profile |
| "Invalid bearer token" | Regenerate token: `openpass mcp-config <agent> --http` |
| "Connection refused" | Ensure server is running on correct port; check firewall |
| "Port already in use" | Use different port: `openpass serve --port 8081` |
| Stdio mode hangs | Ensure no other process is reading from stdin |
| HTTP mode timeout | Check if vault is unlocked; server needs unlocked vault |

**Testing MCP connection:**
```bash
# Test HTTP endpoint
curl -H "Authorization: Bearer $(cat ~/.openpass/mcp-token)" \
     -H "X-OpenPass-Agent: default" \
     http://127.0.0.1:8080/mcp

# Generate config for testing
openpass mcp-config default --http
```

---

## Git Sync Issues

### Symptom: Changes not pushed or pull fails

**Diagnostic steps:**

1. Check git status in vault:
   ```bash
   cd ~/.openpass && git status
   ```

2. View recent commits:
   ```bash
   openpass git log
   ```

3. Check remote configuration:
   ```bash
   cd ~/.openpass && git remote -v
   ```

4. Test connectivity:
   ```bash
   cd ~/.openpass && git fetch origin
   ```

**Common Git issues:**

| Problem | Solution |
|---------|----------|
| "Merge conflict" | Resolve manually in vault directory, then commit |
| "Push rejected" | Pull first: `openpass git pull`, resolve conflicts, then push |
| "No remote configured" | Add remote: `cd ~/.openpass && git remote add origin <url>` |
| "Authentication failed" | Check SSH keys or HTTPS credentials |
| Changes not auto-pushed | Check `auto_push: true` in `config.yaml` |

**Removing accidentally tracked artifacts:**
If sensitive runtime artifacts like `mcp-token` or `.runtime-port` were accidentally committed to your vault repository before they were added to `.gitignore`, you can remove them from the history while keeping the local files:

```bash
cd ~/.openpass
# Remove from git index but keep local file
git rm --cached mcp-token .runtime-port .index
# Commit the removal
git commit -m "Remove sensitive runtime artifacts from tracking"
# Push changes
git push origin main
```

**Manual sync procedure:**
```bash
cd ~/.openpass
git pull origin main
# Resolve any conflicts
git add .
git commit -m "Resolve sync conflicts"
git push origin main
```

---

## Platform-Specific Issues

### macOS

**Keychain access issues:**
- If passphrase caching fails, check Keychain Access app for "OpenPass" entries
- Reset keychain: `openpass lock` then `openpass unlock` (re-creates entry)
- Gatekeeper may block unsigned binaries; allow in System Preferences > Security

**LaunchAgent issues:**
- Check logs: `tail -f ~/Library/Logs/openpass-mcp.log`
- Verify plist syntax: `plutil -lint ~/Library/LaunchAgents/com.example.openpass-mcp.plist`
- Reload agent: `launchctl unload ~/Library/LaunchAgents/com.example.openpass-mcp.plist && launchctl load ~/Library/LaunchAgents/com.example.openpass-mcp.plist`

### Linux

**D-Bus / Secret Service issues:**
- Ensure `gnome-keyring` or `kwallet` is running
- Check D-Bus session: `echo $DBUS_SESSION_BUS_ADDRESS`
- Install required libraries: `libsecret-1-0` (Debian/Ubuntu) or `libsecret` (Arch)

**Systemd service:**
```bash
# Check service status
systemctl --user status openpass-mcp

# View logs
journalctl --user -u openpass-mcp -f
```

**File permissions:**
```bash
# Ensure proper ownership
ls -la ~/.openpass/
# Should be owned by your user, not root
```

### Windows

**Credential Manager issues:**
- Open Credential Manager > Windows Credentials
- Look for "OpenPass" entries
- If missing, run `openpass unlock` to recreate

**PATH issues:**
- Ensure `openpass.exe` directory is in PATH
- Use full path if needed: `C:\Users\You\bin\openpass.exe`

**WSL considerations:**
- WSL uses Linux keyring (D-Bus), not Windows Credential Manager
- Ensure WSL has D-Bus running: `sudo service dbus start`

---

## Performance Issues

### Symptom: Slow vault operations or high memory usage

**Diagnostic steps:**

1. Check vault size:
   ```bash
   du -sh ~/.openpass/
   du -sh ~/.openpass/entries/
   ```

2. Count entries:
   ```bash
   find ~/.openpass/entries -name "*.age" | wc -l
   ```

3. Check for large entries:
   ```bash
   ls -laS ~/.openpass/entries/ | head -20
   ```

4. Monitor system resources:
   ```bash
   # macOS
   top -o cpu | grep openpass
   
   # Linux
   ps aux | grep openpass
   ```

**Optimization tips:**

| Issue | Solution |
|-------|----------|
| Too many entries in root | Organize into subdirectories: `work/`, `personal/` |
| Large entry files | Avoid storing large notes; keep entries focused |
| Slow listing | Use prefix filter: `openpass list work/` instead of `openpass list` |
| High memory | Restart MCP server periodically; check for memory leaks |
| Slow git operations | Exclude large files; use `.gitignore` for non-essential files |

---

## Diagnostic Commands Reference

### Quick status check
```bash
openpass version                    # Show version
openpass --vault ~/.openpass list   # Test vault access
openpass unlock                     # Verify passphrase
```

### MCP diagnostics
```bash
# HTTP mode
curl -s http://127.0.0.1:8080/health
curl -H "Authorization: Bearer $(cat ~/.openpass/mcp-token)" \
     -H "X-OpenPass-Agent: default" \
     http://127.0.0.1:8080/mcp

# Config generation
openpass mcp-config default --http
openpass mcp-config default
```

### Vault diagnostics
```bash
ls -la ~/.openpass/                 # Check vault structure
file ~/.openpass/identity.age       # Verify identity file
openpass git log                    # Check sync status
cat ~/.openpass/config.yaml         # Review configuration
```

### System diagnostics
```bash
# Process check
ps aux | grep openpass

# Port check (macOS/Linux)
lsof -i :8080

# Port check (all platforms)
netstat -an | grep 8080
```

---

## Before You Open an Issue

Please complete this checklist before opening a GitHub issue:

- [ ] I am running the latest version of OpenPass (`openpass version`)
- [ ] I have checked this troubleshooting guide for my issue
- [ ] My vault is initialized and unlocked (`openpass unlock` works)
- [ ] The MCP server is running (for agent issues)
- [ ] I am using the correct agent profile name
- [ ] I have tried the [Quick Fixes](#quick-fixes) section
- [ ] I can reproduce the issue consistently
- [ ] I have included my OS and version (e.g., macOS 14.2, Ubuntu 22.04)
- [ ] I have included output of `openpass version`
- [ ] I have sanitized my config (removed tokens/secrets) if sharing

**Information to include:**
1. OpenPass version
2. Operating system and version
3. Go version (if building from source)
4. Steps to reproduce
5. Expected behavior
6. Actual behavior
7. Relevant error messages (redact sensitive info)
8. Output of diagnostic commands above

**For security issues:**
Do NOT open public issues. Email security@openpass.dev with details.

---

## Related Documentation

- [Agent Integration Guide](agent-integration.md) - MCP setup and configuration
- [Runbook](runbook.md) - Operational procedures and incident response
- [Error Tracking Strategy](error-tracking-strategy.md) - Error handling and reporting
- [README](../README.md) - General usage and installation
