# Storage Design

Understanding Memvault's file structure and database design.

## Directory Structure

```
~/memvault/                    # Vault root (MEMVAULT_HOME)
├── sessions/                  # Conversation archives
│   ├── abc123.../            # Session ID (first 32 chars)
│   │   ├── messages/         # Message files
│   │   │   ├── 0001.json    # User/assistant messages
│   │   │   └── 0002.json
│   │   ├── meta.json        # Session metadata
│   │   └── summary.md       # Human-readable summary
│   └── def456.../
├── memories/                  # Agent memories
│   ├── background/           # Who you are
│   │   └── identity.md
│   ├── preference/           # How you work
│   │   └── code-style.md
│   ├── knowledge/            # Facts learned
│   │   └── project-arch.md
│   └── summary/              # Session summaries
│       └── 2026-10-15.md
└── .memvault/                # Internal data
    ├── db/                   # SQLite databases
    │   ├── sessions.db      # Session index
    │   ├── memories.db      # Memory index
    │   └── fts.db           # Full-text search
    ├── config.yaml          # Configuration
    └── logs/                # Application logs
```

## Session Storage

### Message Files

Each message is one JSON file:

```json
// sessions/abc123/messages/0001.json
{
  "version": "1.0",
  "role": "user",
  "content": "Help me implement folder picker",
  "timestamp": "2026-10-15T08:00:00Z",
  "meta": {
    "model": "claude-opus-5-5",
    "attachments": []
  }
}
```

### Session Metadata

```json
// sessions/abc123/meta.json
{
  "id": "abc123...",
  "agent": "claude",
  "native_id": "4b67a8e4-b697-4ede-...",
  "title": "Folder Picker Implementation",
  "created": "2026-10-15T08:00:00Z",
  "updated": "2026-10-15T10:00:00Z",
  "message_count": 42
}
```

### Why Individual Message Files?

- ✅ Append-only (no rewriting large files)
- ✅ Concurrent writes safe
- ✅ Easy to stream
- ✅ Version control friendly

## Memory Storage

### Markdown Format

```markdown
<!-- memories/preference/code-style.md -->
<!-- version: 5e8f3a2... -->
<!-- parents: 2a1b9c8..., 7d4e1f0... -->
# Code Review Preference

User wants security-focused code reviews with emphasis on:
- Input validation
- SQL injection prevention
- XSS protection

**Why**: Working on security-sensitive project.

**Evidence**:
- Session abc123, message 5
- Session def456, message 12
```

### Frontmatter

```yaml
---
id: 5e8f3a2b...
kind: preference
title: Code Review Preference
updated: 2026-10-15T10:00:00Z
author: claude
---
```

## Database Layer

### SQLite Databases

#### sessions.db

```sql
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  agent TEXT NOT NULL,
  native_id TEXT NOT NULL,
  title TEXT,
  created TIMESTAMP,
  updated TIMESTAMP,
  message_count INTEGER,
  raw_path TEXT -- Path to message files
);

CREATE INDEX idx_sessions_agent ON sessions(agent);
CREATE INDEX idx_sessions_created ON sessions(created);
```

#### memories.db

```sql
CREATE TABLE memories (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,  -- background, preference, knowledge, summary
  title TEXT NOT NULL,
  body TEXT,
  revision TEXT,       -- Content hash for conflict detection
  updated TIMESTAMP,
  author TEXT
);

CREATE INDEX idx_memories_kind ON memories(kind);
CREATE INDEX idx_memories_updated ON memories(updated);
```

#### fts.db (Full-Text Search)

```sql
-- FTS5 virtual table
CREATE VIRTUAL TABLE messages_fts USING fts5(
  session_id,
  message_file,
  role,
  content,
  timestamp,
  tokenize='porter unicode61'
);

CREATE VIRTUAL TABLE memories_fts USING fts5(
  memory_id,
  kind,
  title,
  body,
  tokenize='porter unicode61'
);
```

### Why SQLite?

- ✅ Zero configuration
- ✅ Single file database
- ✅ FTS5 for fast search
- ✅ ACID transactions
- ✅ Embeddable
- ✅ Cross-platform

## Indexing Strategy

### Write Path

```
New Message
    ↓
Write JSON file (atomic)
    ↓
Update sessions.db (metadata)
    ↓
Insert into FTS index (async)
```

### Read Path (Search)

```
Search Query
    ↓
FTS query (fast)
    ↓
Load message files (lazy)
    ↓
Return results
```

### Rebuild Index

```bash
memvault index rebuild
```

Scans all files and rebuilds FTS indices.

## File Format Versioning

### Message Format Version

```json
{
  "version": "1.0",  // Format version
  "role": "user",
  ...
}
```

Future versions can add fields without breaking readers:
- v1.0: Basic message
- v1.1: Add attachments
- v2.0: Breaking change (new required field)

### Memory Format Version

```markdown
<!-- version: 5e8f3a2... -->
```

Content-addressed versioning:
- Hash of content
- Detects conflicts
- Enables CRDT-like merge

## Sync Considerations

### File-Based Sync

Works with:
- iCloud
- Dropbox
- Syncthing
- rsync
- Git

### Conflict Detection

```markdown
<!-- memories/preference/lang.md -->
<<<<<<< HEAD
User prefers Go
=======
User prefers Rust
>>>>>>> Remote
```

Memvault detects conflict markers and flags for review.

### Partial Sync

Only sync directories you need:
```bash
# Just memories
rsync -av ~/memvault/memories/ remote:/memvault/memories/

# Specific sessions
rsync -av ~/memvault/sessions/abc123/ remote:/sessions/abc123/
```

## Backup Strategy

### Full Backup

```bash
tar czf memvault-backup-$(date +%Y%m%d).tar.gz ~/memvault/
```

### Incremental Backup

```bash
# rsync only changed files
rsync -av --update ~/memvault/ /backup/memvault/
```

### Git Backup

```bash
cd ~/memvault
git init
git add .
git commit -m "backup $(date)"
git push backup main
```

## Performance

### Read Performance

| Operation | Typical Time |
|-----------|--------------|
| List sessions | 1-5ms |
| Read session meta | <1ms |
| Load messages | 5-20ms |
| Full-text search | 10-100ms |
| Load memory | 1-5ms |

### Write Performance

| Operation | Typical Time |
|-----------|--------------|
| Append message | 1-5ms |
| Update session meta | 1-5ms |
| Write memory | 1-5ms |
| Rebuild index | 1-10s |

### Storage Size

| Data Type | Size per 1000 |
|-----------|---------------|
| Messages | ~1-5 MB |
| Sessions metadata | ~100 KB |
| Memories | ~100-500 KB |
| FTS index | ~2-10 MB |

## Data Migration

### Export

```bash
# Export all to JSON
memvault export --format json --to /export/

# Export specific session
memvault sessions export abc123 --to session.json
```

### Import

```bash
# Import from export
memvault import --from /export/

# Import single session
memvault sessions import session.json
```

## Future Improvements

### Planned

- Compression for old sessions
- Archive to object storage
- Pluggable storage backends
- Encryption at rest

### Research

- CRDT-based memory merge
- Vector embeddings for semantic search
- Distributed storage

## Best Practices

### Storage Location

- ✅ Use SSD for vault root (fast search)
- ✅ Regular backups to HDD/NAS
- ✅ Sync recent sessions to cloud
- ✅ Archive old sessions offline

### File Organization

- ✅ One session = one directory
- ✅ Immutable message files
- ✅ Mutable memory files
- ✅ Human-readable formats

### Index Maintenance

- ✅ Rebuild after bulk import
- ✅ Rebuild after sync conflicts
- ✅ Automatic rebuild on corruption

## Next Steps

- [Installation](/guide/installation) - Set up vault
- [Archiving](/guide/archiving) - How sessions are captured
- [Search](/guide/search) - Full-text search usage
