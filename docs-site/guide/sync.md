# Multi-Device Sync Guide

**Last Updated**: October 2026 | **Reading Time**: 8 minutes

Learn how to keep Memvault archives synchronized across multiple machines using file-based sync tools. This guide covers setup, conflict resolution, and best practices for multi-device workflows.

## Overview

Memvault uses a **file-based architecture** that works with standard sync tools:
- iCloud Drive
- Dropbox
- Syncthing
- Git
- rsync
- Any file-level sync solution

No custom sync service needed—you control the sync mechanism and schedule.

## Sync Architecture

### File-Based Design

```
Machine A                           Machine B
~/memvault/                         ~/memvault/
├── sessions/       <──Sync Tool──> ├── sessions/
├── memories/                       ├── memories/
└── .memvault/db/                   └── .memvault/db/
```

**What syncs**:
- All session archives (complete)
- All memory files (Markdown)
- Search database (rebuilt locally)

**What doesn't sync**:
- Active sessions (in-progress)
- Temporary files
- Local cache

This aligns with [local-first software principles](https://www.inkandswitch.com/local-first/) where data lives on user devices and syncs peer-to-peer.

## Sync Methods

### Method 1: Cloud Sync (Easiest)

Use commercial cloud storage with native sync clients.

#### iCloud Drive (macOS/iOS)

```bash
# 1. Move vault to iCloud
mv ~/memvault ~/Library/Mobile\ Documents/com~apple~CloudDocs/memvault

# 2. Create symlink
ln -s ~/Library/Mobile\ Documents/com~apple~CloudDocs/memvault ~/memvault

# 3. Configure memvault
export MEMVAULT_HOME=~/memvault
```

**Pros**:
- Automatic sync
- Works across macOS/iOS
- Native encryption

**Cons**:
- Apple ecosystem only
- Network dependent
- Storage limits

#### Dropbox

```bash
# 1. Move to Dropbox
mv ~/memvault ~/Dropbox/memvault

# 2. Symlink
ln -s ~/Dropbox/memvault ~/memvault

# 3. Configure
export MEMVAULT_HOME=~/memvault
```

**Pros**:
- Cross-platform
- Selective sync
- Version history

**Cons**:
- Proprietary
- Storage costs
- Privacy considerations

### Method 2: Git (Developer-Friendly)

Track archives in version control.

```bash
cd ~/memvault
git init
git add sessions/ memories/
git commit -m "Initial archive"

# Add remote
git remote add origin git@github.com:you/memvault-private.git
git push -u origin main

# On another machine
git clone git@github.com:you/memvault-private.git ~/memvault
```

**Sync workflow**:

```bash
# Pull before starting work
cd ~/memvault && git pull

# Push after session ends
cd ~/memvault
git add .
git commit -m "Archive: $(date +%Y-%m-%d)"
git push
```

**Pros**:
- Full version history
- Offline capable
- Audit trail

**Cons**:
- Manual commit/push
- Large repos slow over time
- Requires Git knowledge

**Tip**: Use Git LFS for large attachments:

```bash
cd ~/memvault
git lfs install
git lfs track "sessions/*/attachments/*"
```

### Method 3: Syncthing (Privacy-First)

Peer-to-peer sync without cloud storage.

**Install**:
```bash
# macOS
brew install syncthing

# Linux
apt install syncthing

# Windows
choco install syncthing
```

**Configure**:

1. Start Syncthing: `syncthing` (opens web UI at localhost:8384)
2. Add folder: `~/memvault`
3. Share with other devices
4. Accept on other device

**Pros**:
- No cloud needed
- End-to-end encrypted
- Cross-platform
- Continuous sync

**Cons**:
- Requires setup on each device
- Both devices must be online
- More complex than cloud sync

### Method 4: rsync (Manual/Scheduled)

Simple one-way sync for backups or specific workflows.

```bash
# Push to NAS
rsync -av --delete ~/memvault/ nas.home:/backup/memvault/

# Pull from server
rsync -av --delete server:/data/memvault/ ~/memvault/
```

**Automated with launchd (macOS)**:

```xml
<!-- ~/Library/LaunchAgents/com.memvault.sync.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" ...>
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.memvault.sync</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/rsync</string>
        <string>-av</string>
        <string>--delete</string>
        <string>/Users/you/memvault/</string>
        <string>nas:/backup/memvault/</string>
    </array>
    <key>StartInterval</key>
    <integer>3600</integer>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.memvault.sync.plist
```

**Pros**:
- Simple
- Fast
- Scriptable

**Cons**:
- One-way only
- Manual conflict resolution
- No automatic merge

## Conflict Resolution

### Understanding Conflicts

Conflicts occur when:
- Same file edited on both machines
- Sync happens while file is being written
- Merge timing issues

Memvault detects conflicts in:
- Session metadata
- Memory files
- Message files (rare)

### Automatic Conflict Resolution

**Messages**: Never conflict (append-only)

**Sessions**: Memvault merges automatically:
- Combines message lists
- Takes latest metadata
- Preserves all content

**Memories**: Require manual resolution:

```markdown
<!-- memories/preference/code-style.md -->
<<<<<<< Machine A
User prefers tabs for indentation
=======
User prefers spaces (2 width)
>>>>>>> Machine B
```

### Resolving Memory Conflicts

```bash
# List conflicts
memvault memories conflicts

# View conflicted memory
memvault memories read <id>

# Edit to resolve
vim ~/memvault/memories/preference/code-style.md
```

**Resolution strategies**:

1. **Choose one**: Remove conflict markers, keep one version
2. **Merge both**: Combine information if both are valid
3. **Add context**: Note which is newer or more accurate

Example resolution:

```markdown
User prefers spaces (2 width) for indentation.

**Changed**: 2026-10-15 (previously used tabs)
**Reason**: Team standard at new company
```

Mark as resolved:

```bash
memvault memories resolve <id>
```

### Preventing Conflicts

1. **Sync before starting work**:
```bash
# With Dropbox/iCloud: wait for sync to complete
# With Git:
cd ~/memvault && git pull
```

2. **Sync after finishing**:
```bash
# With Git:
cd ~/memvault
git add .
git commit -m "Session: $(date)"
git push
```

3. **Use one machine at a time** for active sessions

4. **Let archives settle** before switching machines

## Best Practices

### 1. Choose One Primary Sync Method

Don't mix sync tools for the same vault:
- ❌ iCloud + Dropbox simultaneously
- ✅ iCloud for personal, Git for work

### 2. Verify Sync Before Acting

```bash
# Check last sync time
ls -lt ~/memvault/sessions/ | head

# Verify no conflicts
memvault memories conflicts
```

### 3. Handle Large Archives

For 100,000+ sessions:

**Option A**: Sync only recent data

```bash
# Sync last 90 days only
rsync -av --delete \
  --exclude="sessions/2025-*" \
  --exclude="sessions/2024-*" \
  ~/memvault/ laptop:/memvault/
```

**Option B**: Archive old data

```bash
# Move old sessions to separate vault
memvault sessions archive \
  --older-than 730d \
  --to ~/memvault-archive/
```

### 4. Monitor Sync Health

```bash
# Check for incomplete syncs
find ~/memvault/sessions -name "*.tmp" -o -name ".sync*"

# Verify recent activity
memvault sessions list --from $(date -v-7d +%Y-%m-%d)
```

### 5. Backup Before Bulk Operations

```bash
# Before merge/archive/cleanup
tar czf ~/memvault-backup-$(date +%Y%m%d).tar.gz ~/memvault/
```

## Platform-Specific Tips

### macOS

**iCloud Status**:
```bash
brctl log --wait --shorten
```

**Force sync**:
```bash
# Trigger immediate sync
killall bird
```

### Windows

**Dropbox**:
```powershell
# Check sync status
Get-Process | Where-Object {$_.Name -eq "Dropbox"}
```

### Linux

**Systemd timer for rsync**:

```ini
# ~/.config/systemd/user/memvault-sync.service
[Unit]
Description=Memvault Sync

[Service]
Type=oneshot
ExecStart=/usr/bin/rsync -av --delete %h/memvault/ nas:/backup/memvault/
```

```ini
# ~/.config/systemd/user/memvault-sync.timer
[Unit]
Description=Memvault Sync Timer

[Timer]
OnBootSec=5min
OnUnitActiveSec=1h

[Install]
WantedBy=timers.target
```

```bash
systemctl --user enable memvault-sync.timer
systemctl --user start memvault-sync.timer
```

## Mobile Access (iOS/Android)

### Read-Only Access

For viewing archives on mobile:

**iOS**: Use iCloud Drive + any text/markdown viewer  
**Android**: Use Dropbox + markdown reader

Mobile clients coming in future release.

### Web Interface

Access from any device:

```bash
# On your desktop/server
memvault serve --host 0.0.0.0 --port 27961

# From phone browser
http://your-machine-ip:27961
```

**Security**: Only expose on trusted networks or use SSH tunnel:

```bash
# From phone (with SSH client)
ssh -L 27961:localhost:27961 user@desktop

# Then browse to localhost:27961
```

## Troubleshooting

### Sync Loops

**Symptom**: Files constantly syncing back and forth

**Cause**: Timestamp issues or database regeneration

**Fix**:

```bash
# Stop memvault
memvault stop

# Delete local cache
rm -rf ~/.memvault/cache/

# Let sync complete

# Rebuild index once
memvault index rebuild
```

### Partial Sync

**Symptom**: Some sessions missing on secondary machine

**Causes**:
- Sync in progress
- Storage full
- Selective sync enabled

**Fix**:

```bash
# Check disk space
df -h ~/memvault

# Force full sync (Dropbox)
# Open Dropbox preferences → Sync → Selective Sync → Ensure all folders checked

# Verify session count matches
memvault sessions list | wc -l
```

### Performance Issues

**Symptom**: Slow sync, high CPU/disk usage

**Causes**:
- Many small files
- Large attachments
- Frequent changes

**Optimize**:

```bash
# Consolidate attachments
memvault optimize --pack-attachments

# Archive old sessions
memvault sessions archive --older-than 365d

# Use Git LFS for large files
cd ~/memvault
git lfs track "*/attachments/*.{png,jpg,pdf}"
```

## Next Steps

- **[Archiving Guide](/guide/archiving)** - Optimize what gets synced
- **[Configuration Reference](/reference/configuration)** - Sync-related settings
- **[CLI Reference](/reference/cli)** - Session management commands

## Sources

- [Local-First Software Principles](https://www.inkandswitch.com/local-first/)
- [Syncthing Documentation](https://docs.syncthing.net/)
- [Git LFS for Large Files](https://git-lfs.github.com/)

---

**Last reviewed**: October 2026. [Contribute improvements](https://github.com/liyown/memvault/blob/main/CONTRIBUTING.md).
