# Data Sovereignty

Understanding ownership and control of your AI conversation data.

## What is Data Sovereignty?

**Data sovereignty** means you have complete legal and technical control over your data.

With Memvault:
- **You own** the files
- **You control** who accesses them
- **You decide** where they're stored
- **You choose** how to sync them

## Legal Ownership

### Cloud Services (ChatGPT, Claude.ai)

When you use cloud AI platforms:

```
┌─────────────────────────────────────┐
│  Their Servers                      │
│  ├── Your conversations             │
│  ├── Their terms of service         │
│  └── Their data retention policy    │
└─────────────────────────────────────┘
```

- Platform owns the infrastructure
- You have limited export rights
- Data subject to their policies
- Changes affect your access

### Memvault (Local-First)

```
┌─────────────────────────────────────┐
│  Your Computer / Your Rules         │
│  ├── Your conversations (owned)     │
│  ├── Your sync choice               │
│  └── Your retention policy          │
└─────────────────────────────────────┘
```

- You own the files
- Full export rights (it's already exported)
- Data subject to your rules
- No platform can change access

## Technical Control

### What You Can Do

**Read**: Direct file access
```bash
cat ~/memvault/sessions/abc123/messages/0001.json
```

**Search**: Your own index
```bash
grep -r "search term" ~/memvault/sessions/
```

**Process**: Any tool
```bash
jq '.messages[] | select(.role=="user")' session.json
```

**Backup**: Your method
```bash
rsync -av ~/memvault/ /backup/memvault/
```

**Encrypt**: Your keys
```bash
# Encrypt entire vault
tar czf - ~/memvault | gpg -e > vault.tar.gz.gpg
```

**Delete**: Actually delete
```bash
rm -rf ~/memvault/sessions/abc123/
# It's gone. Really gone.
```

## Compliance Scenarios

### GDPR (Europe)

**Right to erasure**: Delete files immediately
```bash
memvault sessions delete <id>
rm ~/memvault/sessions/<id>/
```

**Data portability**: Already in open format
```bash
cp -r ~/memvault/ /export/
```

**Right to know**: Audit trail in files
```bash
ls -la ~/memvault/sessions/<id>/
```

### CCPA (California)

**Know what's collected**: Direct file inspection
**Delete on request**: Instant, verifiable
**No sale of data**: It never leaves your machine

### Corporate Policies

**Data residency**: Files stay where you put them
**Access control**: OS-level permissions
**Audit trail**: File metadata + git history
**Retention**: Your policy, enforced locally

## Multi-Device Sovereignty

### You Control Sync

Memvault never syncs for you. You choose:

**Option 1: Cloud Storage**
```bash
# iCloud
~/Library/Mobile Documents/com~apple~CloudDocs/memvault/

# Dropbox
~/Dropbox/memvault/
```
→ Data passes through their servers  
→ Subject to their terms  
→ Encrypted in transit  

**Option 2: Self-Hosted**
```bash
# Your NAS
rsync -av ~/memvault/ nas.home:/memvault/

# Your server
git remote add origin ssh://your-server/memvault.git
```
→ Your infrastructure  
→ Your rules  
→ Full control  

**Option 3: Peer-to-Peer**
```bash
# Syncthing
syncthing add ~/memvault/
```
→ No server  
→ Direct device-to-device  
→ Complete sovereignty  

**Option 4: Air-Gapped**
```bash
# USB key
cp -r ~/memvault/ /Volumes/USB/memvault/
```
→ Physically isolated  
→ Maximum security  
→ Manual sync  

### Conflict Resolution

You own conflicts too:

```markdown
<!-- memories/preference/lang.md -->
<<<<<<< HEAD
User prefers Go
=======
User prefers Rust
>>>>>>> Synced from laptop
```

Memvault flags but doesn't auto-resolve—you decide.

## Comparison with Cloud Services

| Aspect | Cloud AI | Memvault |
|--------|----------|----------|
| **File ownership** | Platform | You |
| **Storage location** | Their servers | Your choice |
| **Access control** | Platform policy | OS permissions |
| **Deletion** | "Soft delete" | Real delete |
| **Export** | Limited API | Direct access |
| **Encryption** | Their keys | Your keys |
| **Sync** | Automatic | Your choice |
| **Terms** | Theirs | Yours |

## Practical Sovereignty

### Scenario 1: Corporate Requirement

> "All AI conversation data must stay on-premises."

✅ **Memvault**: Install on corporate machines, sync to company NAS  
❌ **Cloud AI**: Violates policy  

### Scenario 2: Sensitive Research

> "Our research data can't touch third-party servers."

✅ **Memvault**: Air-gapped machines, USB sync  
❌ **Cloud AI**: Not an option  

### Scenario 3: Legal Hold

> "We need to preserve all AI interactions for litigation."

✅ **Memvault**: Copy files, immutable archives  
❌ **Cloud AI**: Request export, hope it's complete  

### Scenario 4: Right to Be Forgotten

> "A user requests deletion of their data."

✅ **Memvault**: Delete files, verify with `ls`  
❌ **Cloud AI**: Request deletion, trust platform  

## Long-Term Sovereignty

### Platform Changes

**Cloud AI Platform**:
- Changes terms → You must accept or lose access
- Changes pricing → You pay or export
- Changes export → You adapt to new format
- Shuts down → You scramble to export

**Memvault**:
- You control files → Changes don't affect ownership
- No pricing → It's your disk
- No export changes → Already exported
- You control lifecycle → Independent of any platform

### Future-Proofing

Your data outlives any platform:

```bash
# 2026: Today
memvault search "query"

# 2036: Still works
grep -r "query" ~/memvault/

# 2046: Still works
cat ~/memvault/sessions/*/messages/*.json
```

Open formats guarantee access forever.

## Encryption and Security

### At Rest

Your responsibility:
```bash
# Full disk encryption (macOS)
FileVault

# Folder encryption
gpg --encrypt-files ~/memvault/

# Container encryption
VeraCrypt, LUKS
```

### In Transit

If syncing:
```bash
# iCloud: Encrypted in transit
# Dropbox: Encrypted in transit
# Syncthing: Encrypted by default
# rsync over SSH: Encrypted
```

### At Rest on Sync Service

If using cloud sync:
- **iCloud**: Encrypted with your Apple ID
- **Dropbox**: Encrypted on their servers
- **Google Drive**: Encrypted on their servers
- **Self-hosted**: Your encryption

## Best Practices

### 1. Choose Appropriate Sync

- Sensitive data → Self-hosted or P2P
- Personal use → iCloud / Dropbox OK
- Corporate → On-premises only
- Air-gapped → USB / offline

### 2. Regular Backups

```bash
# Daily backup
rsync -av ~/memvault/ /backup/$(date +%Y%m%d)/
```

### 3. Access Control

```bash
# Restrict vault permissions
chmod 700 ~/memvault/
```

### 4. Audit Trail

```bash
# Version control
cd ~/memvault && git init
git add . && git commit -m "daily snapshot"
```

## Summary

**Data sovereignty means**:
- ✅ Legal ownership
- ✅ Technical control
- ✅ Policy compliance
- ✅ Future independence

**Memvault provides**:
- ✅ Local storage
- ✅ Open formats
- ✅ Direct access
- ✅ Your choice of sync

**You retain**:
- ✅ Complete control
- ✅ No vendor lock-in
- ✅ Privacy by design
- ✅ Long-term access

## Next Steps

- [Storage Design](/concepts/storage) - File structure details
- [Multi-Device Sync](/guide/sync) - Sync strategies
- [Configuration](/reference/configuration) - Security settings
