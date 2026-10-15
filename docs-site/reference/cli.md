# CLI Reference

Complete command-line reference for Memvault.

## Global Options

```bash
memvault [--home DIR] [--log-level LEVEL] <command> [options]
```

| Option | Description | Default |
|--------|-------------|---------|
| `--home` | Vault directory | `$MEMVAULT_HOME` or `~/memvault` |
| `--log-level` | Log verbosity: debug, info, warn, error | `info` |
| `--help` | Show help | |
| `--version` | Show version | |

## Commands

### `memvault serve`

Start HTTP server for web UI and MCP.

```bash
memvault serve [options]
```

**Options**:

| Option | Description | Default |
|--------|-------------|---------|
| `--host` | Bind address | `127.0.0.1` |
| `--port` | Port number | `27961` |
| `--open` | Open browser | `false` |
| `--no-hooks` | Disable hook endpoints | `false` |

**Examples**:

```bash
# Start server
memvault serve

# Bind to all interfaces
memvault serve --host 0.0.0.0

# Custom port
memvault serve --port 8080

# Start and open browser
memvault serve --open
```

### `memvault sessions`

Manage conversation sessions.

#### `sessions list`

```bash
memvault sessions list [options]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--agent <name>` | Filter by agent |
| `--from <date>` | Start date (YYYY-MM-DD) |
| `--to <date>` | End date (YYYY-MM-DD) |
| `--limit <n>` | Max results |
| `--sort <field>` | Sort by: created, updated, messages |
| `--format <fmt>` | Output format: table, json, csv |

**Examples**:

```bash
# List all sessions
memvault sessions list

# Filter by agent
memvault sessions list --agent claude

# Recent sessions
memvault sessions list --from 2026-10-01

# Export to JSON
memvault sessions list --format json > sessions.json
```

#### `sessions read`

```bash
memvault sessions read <session-id> [options]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--format <fmt>` | Output: text, json, markdown |
| `--limit <n>` | Max messages |
| `--offset <n>` | Skip messages |
| `--full` | Include attachments |

**Examples**:

```bash
# Read session
memvault sessions read abc123

# Export to markdown
memvault sessions read abc123 --format markdown > session.md

# Read with pagination
memvault sessions read abc123 --limit 50 --offset 100
```

#### `sessions delete`

```bash
memvault sessions delete <session-id> [--force]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--force` | Skip confirmation |

#### `sessions link`

```bash
memvault sessions link --from <id> --to <id> --type <type>
```

**Types**: `continuation`, `branch`, `reference`

**Example**:

```bash
memvault sessions link \
  --from old-session \
  --to new-session \
  --type continuation
```

#### `sessions archive`

```bash
memvault sessions archive [options]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--older-than <duration>` | Archive sessions older than (e.g., 365d, 2y) |
| `--to <dir>` | Target directory |
| `--dry-run` | Show what would be archived |

**Example**:

```bash
# Archive sessions older than 2 years
memvault sessions archive \
  --older-than 730d \
  --to ~/memvault-archive/
```

### `memvault search`

Search conversations and memories.

```bash
memvault search <query> [options]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--agent <name>` | Filter by agent |
| `--kind <type>` | Search: all, session, memory |
| `--from <date>` | Start date |
| `--to <date>` | End date |
| `--limit <n>` | Max results |
| `--context <n>` | Messages before/after |
| `--format <fmt>` | Output: text, json, csv |

**Examples**:

```bash
# Simple search
memvault search "folder picker"

# With filters
memvault search "implementation" \
  --agent claude \
  --from 2026-10-01

# Search memories only
memvault search "preferences" --kind memory

# Export results
memvault search "bug fix" --format json > results.json
```

### `memvault memories`

Manage semantic memories.

#### `memories list`

```bash
memvault memories list [options]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--kind <type>` | Filter: background, preference, knowledge, summary |
| `--sort <field>` | Sort: created, updated, title |
| `--format <fmt>` | Output format |

#### `memories read`

```bash
memvault memories read <memory-id>
```

#### `memories conflicts`

```bash
memvault memories conflicts
```

List memories with unresolved conflicts.

#### `memories resolve`

```bash
memvault memories resolve <memory-id>
```

Mark conflict as resolved after manual edit.

### `memvault index`

Manage search index.

#### `index rebuild`

```bash
memvault index rebuild [--force]
```

Rebuild FTS5 search index from scratch.

#### `index stats`

```bash
memvault index stats
```

Show index statistics and performance metrics.

#### `index update`

```bash
memvault index update
```

Incrementally update index with new content.

### `memvault db`

Database maintenance.

#### `db optimize`

```bash
memvault db optimize
```

Vacuum and optimize database files.

#### `db stats`

```bash
memvault db stats
```

Show database size and statistics.

### `memvault hook`

MCP hook handlers (called by agents).

#### `hook session-start`

```bash
memvault hook session-start [--agent <name>]
```

Handles session initialization hook.

#### `hook user-prompt-submit`

```bash
memvault hook user-prompt-submit [--agent <name>]
```

Captures user message hook.

### `memvault mcp`

MCP server for agent integration.

```bash
memvault mcp [--check]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--check` | Verify MCP configuration |
| `--stdio` | Use stdio transport (default) |

Start MCP server (called by agent):

```bash
memvault mcp
```

### `memvault export`

Export vault for backup or migration.

```bash
memvault export --to <directory> [options]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--to <dir>` | Export directory |
| `--format <fmt>` | Format: native, json, markdown |
| `--from <date>` | Start date |
| `--to-date <date>` | End date |

**Example**:

```bash
# Export everything
memvault export --to /tmp/export/

# Export recent sessions only
memvault export \
  --to /tmp/export/ \
  --from 2026-10-01
```

### `memvault import`

Import from export or other sources.

```bash
memvault import --from <directory> [options]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--from <dir>` | Import directory |
| `--format <fmt>` | Format: native, json, markdown |
| `--dry-run` | Show what would be imported |

### `memvault verify`

Verify vault integrity.

```bash
memvault verify [--check-integrity] [--fix]
```

**Options**:

| Option | Description |
|--------|-------------|
| `--check-integrity` | Deep verification |
| `--fix` | Attempt automatic repair |

### `memvault stats`

Show vault statistics.

```bash
memvault stats [--from <date>] [--to <date>]
```

**Example output**:

```
Vault Statistics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Sessions:           5,432
Messages:          245,678
Memories:              156
  Background:           12
  Preference:           45
  Knowledge:            78
  Summary:              21

Storage:
  Sessions:        2.3 GB
  Attachments:     1.1 GB
  Database:       45.2 MB
  Total:          3.4 GB

Activity (Last 30 Days):
  Sessions:           142
  Messages:         8,765
  Search queries:     523
```

## Configuration File

Default location: `~/.memvault/config.yaml`

Override with: `--home ~/custom-vault`

Example config:

```yaml
server:
  host: 127.0.0.1
  port: 27961

index:
  tokenizer: unicode61
  max_results: 100

sync:
  enabled: false
  method: git
  remote: origin
  auto_push: false

retention:
  sessions_days: 0  # 0 = keep forever
  summaries_days: 0
  temp_files_hours: 24
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MEMVAULT_HOME` | Vault directory | `~/memvault` |
| `MEMVAULT_AGENT_HOME` | Agent install directory | `~/.memvault/agents` |
| `MEMVAULT_LOG_LEVEL` | Log level | `info` |
| `MEMVAULT_PORT` | Server port | `27961` |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Not found (session, memory, etc.) |
| 4 | Permission denied |
| 5 | Database error |
| 10 | Network error |

## Examples

### Daily Backup

```bash
#!/bin/bash
# backup.sh - Daily memvault backup

DATE=$(date +%Y-%m-%d)
BACKUP_DIR=~/backups/memvault

# Export recent sessions
memvault export \
  --to "$BACKUP_DIR/$DATE" \
  --from $(date -v-7d +%Y-%m-%d)

# Compress
tar czf "$BACKUP_DIR/memvault-$DATE.tar.gz" \
  "$BACKUP_DIR/$DATE"

# Clean up
rm -rf "$BACKUP_DIR/$DATE"

# Keep last 30 days
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +30 -delete
```

### Search and Export

```bash
# Find all security discussions
memvault search "security" --format json | \
  jq -r '.results[].session_id' | \
  while read sid; do
    memvault sessions read "$sid" --format markdown \
      > "security-$sid.md"
  done
```

### Stats Dashboard

```bash
#!/bin/bash
# dashboard.sh - Show vault dashboard

echo "=== Memvault Dashboard ==="
echo

echo "Recent Activity:"
memvault sessions list --limit 5 --sort updated

echo
echo "Search Performance:"
memvault index stats | grep "Avg query time"

echo
echo "Storage:"
du -sh ~/memvault/

echo
echo "Today's Sessions:"
memvault sessions list --from $(date +%Y-%m-%d) | wc -l
```

## Next Steps

- **[MCP Tools Reference](/reference/mcp-tools)** - Agent integration
- **[Configuration Reference](/reference/configuration)** - Detailed settings
- **[API Reference](/reference/api)** - HTTP API documentation

---

**Last updated**: October 2026
