# OpenPass Launch & Incident Response Runbook

**Maintainer**: OpenPass Team
**Security Contact**: security@openpass.dev
**Repository**: https://github.com/danieljustus/OpenPass

## Table of Contents

1. [CI/CD Overview](#cicd-overview)
2. [govulncheck Failures](#govulncheck-failures)
3. [Dependabot Alerts](#dependabot-alerts)
4. [Release Workflow Errors](#release-workflow-errors)
5. [Backup & Recovery](#backup--recovery)
6. [Security Incident Response](#security-incident-response)
7. [Post-Release Checklist](#post-release-checklist)

---

## CI/CD Overview

OpenPass uses GitHub Actions for CI/CD:

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yml` | Push to main, PRs | Tests, lint, govulncheck, builds |
| `release.yml` | Version tags (`v*`) | Release artifacts, checksums |

### CI Pipeline Jobs

1. **govulncheck** - Vulnerability scanning
2. **lint** - Code quality checks (golangci-lint)
3. **test** - Unit and integration tests
4. **build** - Cross-platform binary builds
5. **integration-test** - Integration test suite

### Release Pipeline Jobs

1. **test** - Pre-release validation
2. **govulncheck** - Final vulnerability scan
3. **release** - GoReleaser artifact publishing

---

## govulncheck Failures

### What This Means

`govulncheck` found vulnerabilities in OpenPass dependencies (not in OpenPass code itself).

### Response Matrix

| Severity | CVSS | Action | SLA |
|----------|------|--------|-----|
| Critical | 9.0-10.0 | Patch release within 48h | Immediate |
| High | 7.0-8.9 | Patch release within 7 days | 48 hours |
| Medium | 4.0-6.9 | Next minor release | 2 weeks |
| Low | 0.1-3.9 | Next minor release | Next cycle |

### Investigation Steps

1. **Identify the vulnerable package**:
   ```bash
   # Local check
   go run golang.org/x/vuln/cmd/govulncheck@latest ./...

   # Check which package has the CVE
   ```

2. **Check if the vulnerable function is actually called**:
   - govulncheck may report false positives
   - Verify the call chain reaches the vulnerable code path

3. **Check for fixed version**:
   ```bash
   go list -m -versions <package>
   go get <package>@<fixed-version>
   ```

4. **If no fix available**:
   - Monitor the dependency for updates
   - Consider vendoring or forking if critical
   - Document in issue tracker

### Update Process

```bash
# Update dependency
go get <package>@latest
go mod tidy
go mod verify

# Test thoroughly
go test -v ./...
GOWORK=off go test -v -tags smoke ./...

# Commit with conventional message
git commit -m "fix(deps): update <package> to fix CVE-XXXX-XXXX"
```

---

## Dependabot Alerts

### What This Means

Dependabot detected outdated dependencies with known vulnerabilities.

### Response Steps

1. **Review the alert** in GitHub → Security → Dependabot alerts

2. **Check if update is safe**:
   - Major version updates may have breaking changes
   - Check changelog for breaking changes
   - Test locally first

3. **Merge or dismiss**:
   - **Safe to merge**: Click "Merge pull request"
   - **False positive**: Click "Dismiss" with reason
   - **Needs investigation**: Comment and assign to maintainer

4. **For security updates**: Prioritize merging within 24-48 hours

### Handling Major Updates

```bash
# Create feature branch
git checkout -b dependabot/<package>-<version>

# Check for breaking changes
go mod why <package>  # Why is this dependency needed?

# Test with new version
go get <package>@latest
go mod tidy
```

---

## Release Workflow Errors

### Common Failure Points

| Job | Common Failure | Resolution |
|-----|---------------|------------|
| test | Flaky integration test | Re-run, check test isolation |
| govulncheck | New CVE published | Update dependency to fixed version |
| release | GitHub token issues | Verify workflow permissions |
| goreleaser | Asset size limit | Check artifact sizes |

### GoReleaser Failures

1. **Check .goreleaser.yaml** syntax
2. **Verify workflow permissions** include `contents: write`
3. **Check artifact sizes** - GitHub has 10GB total limit
4. **Check GitHub release publishing** if asset upload fails

### Re-running Release

```bash
# Push the tag again (if no changes needed)
git push origin v1.x.x

# Or create a release manually if CI is broken
# See: https://docs.github.com/en/repositories/releasing-projects-on-github/creating-releases
```

---

## Backup & Recovery

### Vault Backup Strategy

OpenPass stores all data in a vault directory. The vault contains:

```
<vault>/
├── identity.age      # Your encrypted age identity (private key)
├── config.yaml       # Vault configuration
├── mcp-token         # Bearer token for HTTP MCP (auto-generated)
├── entries/          # Encrypted password entries (one .age file per entry)
│   ├── github.age
│   └── work/
│       └── aws.age
└── .git/             # Git repository (if enabled)
```

### Backup Methods

| Method | Pros | Cons |
|--------|------|------|
| **Git auto-sync** | Automatic, versioned, distributed | Requires remote push, .git can grow large |
| **Manual copy** | Simple, full control | No incremental history, manual effort |
| **age encryption** | Entries are individually encrypted, portable | Must secure identity.age separately |
| **Export to file** | Portable text format | Requires manual re-import |

### Git Auto-Sync Backup (Recommended)

OpenPass commits vault changes automatically. Ensure remote is configured:

```bash
# Check remote configuration
git remote -v
# origin  git@github.com:user/private-vault.git (push)

# Force a new backup commit (even if no changes)
openpass git commit -m "Backup trigger"

# Manual push if auto_push is disabled
openpass git push
```

To enable a remote for an existing vault:

```bash
cd ~/.openpass  # or your vault path
git remote add origin git@github.com:user/backup-repo.git
git push -u origin main
```

### Manual Vault Backup

```bash
# Create timestamped backup
VAULT_DIR=~/.openpass
BACKUP_DIR=~/backups/openpass
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"
tar -czf "$BACKUP_DIR/vault_$TIMESTAMP.tar.gz" -C ~ "$VAULT_DIR"

# Verify backup integrity
tar -tzf "$BACKUP_DIR/vault_$TIMESTAMP.tar.gz" | head -20
```

### Restoring from Backup

```bash
# Extract backup to temporary location
tar -xzf ~/backups/openpass/vault_20260420_120000.tar.gz -C /tmp/

# Verify identity file
ls -la /tmp/.openpass/identity.age

# Move to vault location
mv ~/.openpass ~/.openpass_old
mv /tmp/.openpass ~/

# Verify vault opens
openpass list
```

### Recovery from Git

```bash
# Clone backup repository
git clone git@github.com:user/backup-repo.git ~/.openpass

# Unlock vault
openpass unlock

# Verify entries
openpass list
```

### Emergency Recovery: Identity Loss

If `identity.age` is lost, **there is no recovery**. The identity is the private key for all encrypted entries.

**Prevention**:
1. Keep `identity.age` in a secure location (hardware backup, safety deposit box)
2. Export the age identity:
   ```bash
   # Export identity (creates unencrypted copy - keep secure!)
   age-keygen -o /tmp/identity.pem
   # Convert back to age format if needed
   ```
3. Use a hardware security key for key storage

### Recovery After System Failure

1. Reinstall OpenPass:
   ```bash
   go install github.com/danieljustus/OpenPass@latest
   ```

2. Restore vault from backup (see above)

3. Verify functionality:
   ```bash
   openpass list
   openpass get test-entry.password
   ```

### Disaster Recovery Checklist

| Step | Action | Verification |
|------|--------|--------------|
| 1 | Restore vault directory | `ls -la ~/.openpass/` shows identity.age and entries/ |
| 2 | Unlock vault | `openpass unlock` succeeds |
| 3 | Verify entries | `openpass list` returns expected entries |
| 4 | Test entry retrieval | `openpass get <entry>` returns password |
| 5 | Verify MCP server | `openpass serve --stdio --agent default` starts |

### Backup Rotation

For critical vaults, use the 3-2-1 rule:
- **3** copies of data
- **2** different media types
- **1** offsite location

Example:
```bash
# Daily incremental backup to external drive
rsync -av --delete ~/.openpass/ /Volumes/Backup/openpass/

# Weekly full backup to cloud storage
rclone sync ~/.openpass/ backblaze:openpass-vaults/$(hostname)/
```

---

## Security Incident Response

### Reporting Process

1. **Receive report** via security@openpass.dev or GitHub Security Advisories
2. **Acknowledge within 48 hours**
3. **Assess severity and impact**
4. **Develop mitigation**
5. **Coordinate disclosure**

### Severity Classification

| Severity | Definition | Response Time |
|----------|------------|---------------|
| Critical | RCE, vault decryption bypass | 24 hours |
| High | Privilege escalation, data exfiltration | 48 hours |
| Medium | Information disclosure, DoS | 7 days |
| Low | Security best practice violation | Next release |

### Incident Response Steps

1. **Triage** (0-24h)
   - Confirm the vulnerability exists
   - Assess impact on users
   - Determine affected versions

2. **Containment** (24-48h)
   - Prepare patch or workaround
   - Identify fix in code
   - Test fix thoroughly

3. **Disclosure** (48-72h)
   - Notify maintainers
   - Prepare security advisory
   - Draft GitHub security advisory (draft mode)

4. **Release** (48-72h for critical)
   - Tag patch release
   - Publish security advisory
   - Notify users via GitHub

5. **Post-Incident**
   - Document lessons learned
   - Add regression tests
   - Update documentation if needed

### Patch Release Process

```bash
# Create patch branch from last release tag
git checkout v1.0.0
git checkout -b security-patch-v1.0.x

# Apply fix
git cherry-pick <commit>
# OR manually apply changes

# Test
go test -v ./...

# Tag and push
git tag v1.0.x
git push origin v1.0.x
```

---

## Post-Release Checklist

After every release, verify:

### Artifact Verification

```bash
# 1. Download artifacts from GitHub Releases
# https://github.com/danieljustus/OpenPass/releases/tag/vX.Y.Z

# 2. Verify checksums
sha256sum -c SHA256SUMS
# or on macOS:
shasum -a 256 --check SHA256SUMS

# 3. Verify binary (Linux example)
./openpass-linux-amd64 version
# Should match tag version
```

### Installation Smoke Test

```bash
# 1. Install on clean system or in container
docker run -it ubuntu:latest bash

# 2. Install dependencies
apt-get update && apt-get install -y curl gpg

# 3. Download and verify release
curl -fsSL https://github.com/danieljustus/OpenPass/releases/download/vX.Y.Z/openpass-linux-amd64 -o openpass
curl -fsSL https://github.com/danieljustus/OpenPass/releases/download/vX.Y.Z/checksums.txt -o checksums.txt

# 4. Check binary works
chmod +x openpass
./openpass version

# 5. Initialize test vault
./openpass init /tmp/test-vault
./openpass add test --value "smoke-test"
./openpass get test
./openpass lock

# 6. Cleanup
rm -rf /tmp/test-vault
```

### GitHub Release Page

- [ ] Release title matches tag
- [ ] Description includes changelog
- [ ] All assets present (binaries for each platform)
- [ ] Checksums file present
- [ ] Security advisory linked (if applicable)

---

## Contact & Resources

| Purpose | Contact |
|---------|---------|
| Security Issues | security@openpass.dev |
| General Issues | [GitHub Issues](https://github.com/danieljustus/OpenPass/issues) |
| Release Verification | See post-release checklist above |

### Useful Links

- [Security Policy](SECURITY.md)
- [Release Process Documentation](.github/workflows/release.yml)
- [GoReleaser Configuration](.goreleaser.yaml)
- [Vulnerability Database](https://pkg.go.dev/golang.org/x/vuln)
