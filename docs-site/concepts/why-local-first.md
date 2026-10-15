# Why Local-First?

Memvault is built on **local-first principles**: your data lives on your machine, under your control.

## The Philosophy

### You Own Your Data

AI conversations are intellectual assets:
- Capture your reasoning process
- Document decisions and context
- Contain sensitive information
- Represent hours of work

**They should belong to you, not the platform.**

## Problems with Cloud-First

### 1. Vendor Lock-In

- Data trapped in proprietary formats
- Can't export full history
- Platform changes affect access
- Risk of service shutdown

### 2. Privacy Concerns

- Conversations analyzed for training
- No guarantee of deletion
- Third-party access possible
- Data residency unclear

### 3. Limited Control

- Can't customize storage
- Search depends on platform
- No offline access
- Rate limits and quotas

### 4. Portability

- Moving between agents is hard
- Historical context doesn't transfer
- Must rebuild context each time
- Walled gardens

## Local-First Benefits

### 1. Complete Ownership

```
Your Machine
├── All conversations
├── Full search index
├── Agent memories
└── Export formats
```

- No platform can delete your data
- No rate limits
- No usage analytics
- No third-party access

### 2. Privacy by Default

- Data never leaves your machine
- No cloud processing
- No AI training on your conversations
- You control who sees what

### 3. Offline-First

- Works without internet
- Search is instant
- No network latency
- No cloud downtime

### 4. Flexible Sync

**You choose how to sync:**

```bash
# iCloud
ln -s ~/Library/Mobile\ Documents/... ~/memvault

# Dropbox
ln -s ~/Dropbox/memvault ~/memvault

# Git
cd ~/memvault && git init

# Syncthing
syncthing add ~/memvault
```

Memvault never implements sync—you use the tool you already trust.

## Architecture Implications

### No Cloud Dependency

```
┌─────────────────────────────────────┐
│         Your Computer               │
│                                     │
│  Agent ←→ Memvault ←→ SQLite       │
│              ↓                      │
│         Markdown Files              │
└─────────────────────────────────────┘
```

- Agent talks to local process
- Data stored locally
- Search index local
- Sync optional

### Open Formats

**Archives**: JSON + Markdown
- Human-readable
- Version control friendly
- Easy to export
- Future-proof

**Memories**: Plain Markdown
- Edit in any text editor
- Version with git
- Search with grep
- Platform-agnostic

### Portable

Move between:
- Machines (sync files)
- Agents (open formats)
- Operating systems (cross-platform)
- Tools (grep, jq, cat work)

## Real-World Scenarios

### Scenario 1: Company Policy

> "We can't use cloud AI that processes our internal data."

✅ **Solution**: Local archive + company NAS sync  
All processing local, sync via corporate infrastructure.

### Scenario 2: Sensitive Work

> "My conversations contain trade secrets and customer data."

✅ **Solution**: Never leaves your machine  
No risk of data leakage, full audit trail.

### Scenario 3: Long-Term Value

> "What if Claude Code shuts down in 5 years?"

✅ **Solution**: Archives in open formats  
Data remains accessible, searchable, portable.

### Scenario 4: Research

> "I want to analyze my AI interaction patterns."

✅ **Solution**: Direct file access  
Use any tool: Python, SQL, jq, Unix utilities.

## Sync Strategy

### You Decide

Memvault provides the **local store**. You choose sync:

| Tool | Use Case | Setup |
|------|----------|-------|
| **iCloud** | macOS/iOS | Link to iCloud folder |
| **Dropbox** | Cross-platform | Link to Dropbox folder |
| **Git** | Version control | `git init` in vault |
| **Syncthing** | P2P sync | Add vault to Syncthing |
| **rsync** | Manual backup | Script to NAS |
| **None** | Single machine | No setup |

### Conflict Resolution

Because sync is external, conflicts are handled by your sync tool:

```markdown
<!-- iCloud/Dropbox conflict markers -->
<<<<<<< Local
User prefers Go
=======
User prefers Rust
>>>>>>> Remote
```

Memvault detects conflicts and flags them for review.

## Comparison

### Cloud-First (ChatGPT, Claude.ai)

❌ Data on vendor servers  
❌ Proprietary export  
❌ No offline search  
❌ Platform lock-in  
❌ Privacy policy changes  

### Local-First (Memvault)

✅ Data on your machine  
✅ Open formats  
✅ Offline-first  
✅ Platform-agnostic  
✅ You set the rules  

## Trade-Offs

### What You Give Up

- No built-in sync service
- Must manage backups yourself
- Cross-device requires setup
- More disk space used

### What You Gain

- Complete control
- Total privacy
- Unlimited storage
- Custom workflows
- Data portability

## Philosophy in Practice

### Design Decisions

1. **SQLite** not cloud DB → offline, fast, portable
2. **Markdown** not proprietary → human-readable, greppable
3. **Files** not API → direct access, version control
4. **Hooks** not webhooks → local process, no network
5. **MCP** not cloud service → local socket, instant

### Future-Proofing

Memvault's formats will work decades from now:

```bash
# In 2046, these still work:
cat memories/background/*.md
jq '.role' sessions/*/messages/*.json
grep -r "search term" sessions/
```

## Getting Started

See [Installation](/guide/installation) to begin using Memvault locally.

## Next Steps

- [Data Sovereignty](/concepts/data-sovereignty)
- [Storage Design](/concepts/storage)
- [Multi-Device Sync](/guide/sync)
