# Full-Text Search Guide

**Last Updated**: October 2026 | **Reading Time**: 10 minutes

Master Memvault's full-text search capabilities powered by SQLite FTS5. Learn search syntax, optimization techniques, and advanced querying patterns to instantly find conversations across your entire archive.

## Table of Contents

- [Quick Start](#quick-start)
- [Search Syntax](#search-syntax)
- [Advanced Queries](#advanced-queries)
- [Performance Optimization](#performance-optimization)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Basic Search

```bash
# Search all conversations
memvault search "folder picker"

# Search with agent filter
memvault search "implementation" --agent claude

# Search by date range
memvault search "bug fix" --from 2026-10-01 --to 2026-10-15
```

### Search from Web UI

1. Open `http://localhost:27961`
2. Click search icon or press `/`
3. Type your query
4. Results appear instantly

## How It Works

Memvault uses **SQLite FTS5** (Full-Text Search, version 5), which builds an [inverted index](https://thelinuxcode.com/sqlite-full-text-search-fts5-in-practice-fast-search-ranking-and-real-world-patterns/) mapping every word to the documents containing it.

### Architecture

```
Query: "folder picker"
    ↓
Tokenizer splits: ["folder", "picker"]
    ↓
Inverted Index lookup:
  "folder" → [session_abc, session_def, ...]
  "picker" → [session_abc, session_xyz, ...]
    ↓
Intersection: [session_abc]
    ↓
BM25 Ranking
    ↓
Results (sorted by relevance)
```

Real-world performance: **10-100ms** on archives with 10,000+ conversations. According to [FTS5 benchmarks](https://medium.com/@arif.rahman.rhm/stop-using-like-how-we-reduced-search-time-by-99-6-with-sqlite-fts5-99cb136e3d00), this is 99.6% faster than traditional LIKE queries.

## Search Syntax

### Single Terms

```bash
# Find "implementation"
memvault search implementation

# Case-insensitive by default
memvault search IMPLEMENTATION  # Same results
```

### Multiple Terms

```bash
# AND (all terms must match)
memvault search "folder picker"

# Results contain both "folder" AND "picker"
```

### Phrase Search

```bash
# Exact phrase
memvault search '"native dialog picker"'

# Must appear in that exact order
```

### Boolean Operators

```bash
# OR (either term)
memvault search "folder OR directory"

# AND (both terms, explicit)
memvault search "picker AND dialog"

# NOT (exclude term)
memvault search "picker NOT windows"

# Combine operators
memvault search "folder OR directory AND picker NOT test"
```

### Prefix Search

```bash
# Words starting with "impl"
memvault search "impl*"

# Matches: implement, implementation, implicit, etc.
```

### Wildcard Search

```bash
# Middle wildcard (slower)
memvault search "*plementation"

# Better: Use prefix when possible
memvault search "impl*"
```

### Column-Specific Search

```bash
# Search in specific fields
memvault search "role:user picker"

# Search session titles only
memvault search "title:implementation"

# Search message content
memvault search "content:folder"
```

### Proximity Search

```bash
# Terms within N words of each other
memvault search "folder NEAR/5 picker"

# "folder" and "picker" within 5 words
```

## Advanced Queries

### Chinese and English Mixed

FTS5's `unicode61` tokenizer handles multi-language content:

```bash
# Chinese search
memvault search "文件夹选择器"

# Mixed language
memvault search "implement 文件夹 picker"
```

### Filtering by Metadata

```bash
# By agent
memvault search "bug fix" --agent claude

# By date range
memvault search "implementation" \
  --from 2026-09-01 \
  --to 2026-09-30

# By session type
memvault search "review" --type code-review

# Combine filters
memvault search "security" \
  --agent claude \
  --from 2026-10-01 \
  --tag important
```

### Context Window

```bash
# Show 3 messages before/after match
memvault search "error" --context 3

# Show more context for debugging
memvault search "exception" --context 10
```

### Result Limiting

```bash
# Top 10 results
memvault search "picker" --limit 10

# Pagination
memvault search "implementation" --limit 20 --offset 20
```

### Sorting

```bash
# By relevance (default, BM25 score)
memvault search "folder picker"

# By date (newest first)
memvault search "picker" --sort date

# By date (oldest first)
memvault search "picker" --sort date --order asc
```

### Output Formats

```bash
# Default: human-readable
memvault search "picker"

# JSON output
memvault search "picker" --format json

# JSON with full message content
memvault search "picker" --format json --full

# CSV for analysis
memvault search "bug" --format csv > bugs.csv
```

## Search Strategies

### Finding Related Conversations

```bash
# 1. Search for core concept
memvault search "authentication"

# 2. Note session IDs from results

# 3. Find linked sessions
memvault sessions links abc123

# 4. Read full context
memvault sessions read abc123
```

### Debugging Past Issues

```bash
# Find error patterns
memvault search "error OR exception OR failed"

# Filter by time period when bug appeared
memvault search "authentication failed" \
  --from 2026-10-10 \
  --to 2026-10-12

# Check what changed before the error
memvault sessions read <session-before-bug>
```

### Tracking Decisions

```bash
# Find architectural discussions
memvault search "architecture OR design OR pattern"

# Find specific technology choices
memvault search "chose Go OR picked TypeScript OR decided"

# Export for documentation
memvault search "decision" --format json | \
  jq '.results[] | {date, session, decision: .snippet}' > decisions.json
```

### Learning from History

```bash
# How did I solve similar problems?
memvault search "solved OR fixed OR resolved"

# What have I learned about X?
memvault search "learned OR discovered OR found out"

# Review past code reviews
memvault search "code review" --tag review
```

## Performance Optimization

### Index Maintenance

The search index is automatically updated as conversations are captured. For optimal performance:

```bash
# Rebuild index after bulk import
memvault index rebuild

# Optimize index (defragment)
memvault db optimize

# Check index health
memvault index stats
```

### Query Performance Tips

**Fast Queries** (< 10ms):
```bash
# Single term
memvault search implementation

# Prefix search
memvault search impl*

# Two terms AND
memvault search "folder picker"
```

**Slower Queries** (10-100ms):
```bash
# Many OR terms
memvault search "word1 OR word2 OR word3 OR ..."

# Wildcard (not prefix)
memvault search "*mentation"

# Complex proximity
memvault search "word1 NEAR/2 word2 AND word3 NEAR/5 word4"
```

**Avoid** (can be slow on large archives):
```bash
# Leading wildcard
memvault search "*word"

# Very broad OR with no filters
memvault search "a OR the OR is OR and"
```

### Index Statistics

```bash
$ memvault index stats

Index Statistics:
  Total documents: 5,432
  Total tokens: 1,245,678
  Index size: 45.2 MB
  Last rebuild: 2026-10-15 10:30:00
  
Performance:
  Avg query time: 15ms
  95th percentile: 45ms
  99th percentile: 120ms
```

### Large Archive Optimization

For archives with 100,000+ conversations:

```bash
# 1. Split by date
memvault search "recent query" --from 2026-10-01

# 2. Use specific filters
memvault search "query" --agent claude --tag project-a

# 3. Archive old sessions to separate vault
memvault sessions archive --older-than 730d \
  --to ~/memvault-archive-2024/
```

## Troubleshooting

### No Results Found

**Check query syntax**:
```bash
# Wrong: Unmatched quotes
memvault search "picker

# Right: Proper quotes
memvault search "picker"
```

**Verify index**:
```bash
# Check if index is healthy
memvault index stats

# Rebuild if needed
memvault index rebuild
```

**Check filters**:
```bash
# Remove filters to broaden search
memvault search "picker"  # No filters

# vs
memvault search "picker" --agent codex  # Might be too narrow
```

### Slow Searches

**Profile query**:
```bash
# Enable timing
memvault search "query" --verbose

Output:
  Parse: 0.2ms
  Index lookup: 45ms
  Load messages: 12ms
  Ranking: 3ms
  Total: 60.2ms
```

**Optimize**:
```bash
# Rebuild index
memvault index rebuild

# Optimize database
memvault db optimize

# Check disk speed
time find ~/memvault/sessions/ -type f | wc -l
```

### Index Corruption

**Symptoms**:
- Search returns errors
- Results are incomplete
- Queries time out

**Fix**:
```bash
# Backup current index
cp -r ~/.memvault/db/ ~/.memvault/db-backup/

# Rebuild from scratch
memvault index rebuild --force

# Verify
memvault search "test query"
```

### Missing Recent Conversations

**Cause**: Index update lag

**Solution**:
```bash
# Force index update
memvault index update

# Or rebuild
memvault index rebuild
```

## Search API

### Programmatic Search

```bash
# JSON output for scripting
curl -s http://localhost:27961/api/v1/search \
  -d '{"query":"picker","limit":10}' | jq .
```

```json
{
  "results": [
    {
      "session_id": "abc123",
      "message_id": "0042",
      "score": 0.95,
      "snippet": "...implement folder **picker** with native...",
      "timestamp": "2026-10-15T08:23:45Z"
    }
  ],
  "total": 42,
  "query_time_ms": 15
}
```

### Python Integration

```python
import subprocess
import json

def search_memvault(query, limit=10):
    result = subprocess.run(
        ["memvault", "search", query, "--format", "json", "--limit", str(limit)],
        capture_output=True,
        text=True
    )
    return json.loads(result.stdout)

# Use it
results = search_memvault("folder picker")
for hit in results["results"]:
    print(f"{hit['timestamp']}: {hit['snippet']}")
```

## Next Steps

- **[Agent Memory Guide](/guide/memory)** - Let agents learn from search results
- **[Archiving Guide](/guide/archiving)** - Optimize what gets indexed
- **[CLI Reference](/reference/cli)** - Complete search command documentation
- **[Configuration](/reference/configuration)** - Advanced search settings

## Sources

- [SQLite FTS5: Fast Search, Ranking, and Patterns](https://thelinuxcode.com/sqlite-full-text-search-fts5-in-practice-fast-search-ranking-and-real-world-patterns/)
- [Stop Using LIKE: 99.6% Faster with FTS5](https://medium.com/@arif.rahman.rhm/stop-using-like-how-we-reduced-search-time-by-99-6-with-sqlite-fts5-99cb136e3d00)
- [Fast Full-Text Search with SQLite FTS5](https://github.com/mneves75/ffts-grep)
- [Optimizing FTS Performance in SQLite](https://www.slingacademy.com/article/optimizing-full-text-search-performance-in-sqlite/)
- [SQLite FTS5 Extension Documentation](https://www.sqlite.org/fts5.html)

---

**Last reviewed**: October 2026 by the Memvault team. [Contribute improvements](https://github.com/liyown/memvault/blob/main/CONTRIBUTING.md).
