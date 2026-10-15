# Agent Memory System Guide

**Last Updated**: October 2026 | **Reading Time**: 15 minutes

Learn how to give AI agents persistent memory across sessions using Memvault's MCP integration. This guide covers memory architecture, agent workflows, and best practices for maintaining agent context.

## Overview

AI agents are stateless by default—they forget everything between sessions. Memvault's memory system solves this by:

1. **Capturing conversations** as they happen
2. **Extracting semantic facts** from discussions  
3. **Providing context** to agents via MCP protocol
4. **Enabling continuity** across sessions and agents

According to [Oracle's research on multi-tenant agent memory](https://blogs.oracle.com/developers/context-assembly-for-multi-tenant-agent-memory-budgets-selection-and-tool-calls-with-mcp), proper memory management requires treating memory as first-class infrastructure with explicit lifecycle policies.

## Memory Architecture

### Two-Layer System

```
┌────────────────────────────────────┐
│  Agent Session                     │
└──────────┬─────────────────────────┘
           │
     ┌─────▼──────┐       ┌──────────┐
     │  Archives  │       │ Memories │
     │ (Complete) │       │ (Semantic)│
     └────────────┘       └──────────┘
          │                     │
    Full history          Key facts
    Immutable            Mutable
    Large files          Small files
    Search index         Direct load
```

**Archives**: Complete conversation records (see [Archiving Guide](/guide/archiving))  
**Memories**: Extracted facts maintained by agents

This separation follows [memory management best practices](https://github.com/doobidoo/mcp-memory-service/wiki/) for production AI systems.

## Memory Types

### 1. Background

Who you are, your expertise, your context.

```markdown
<!-- memories/background/identity.md -->
# User Identity

Software engineer specializing in distributed systems.
Primary languages: Go, TypeScript, Rust.
Currently working on: Local-first application architecture.
```

**Updated**: Rarely (only when facts change)  
**Used by**: All agents, all sessions

### 2. Preference

How you like to work, coding styles, review criteria.

```markdown
<!-- memories/preference/code-review.md -->
# Code Review Preferences

User wants security-focused reviews emphasizing:
- Input validation at all boundaries
- Prepared statements for SQL
- Context timeouts in Go

**Why**: Working on security-sensitive systems.

**Evidence**: Session abc123, message 12
```

**Updated**: When preferences change or new ones are discovered  
**Used by**: Agents making decisions on your behalf

### 3. Knowledge

Facts learned from conversations.

```markdown
<!-- memories/knowledge/project-arch.md -->
# Memvault Architecture

This project uses:
- Go backend with embed for web assets
- React frontend with Vite
- SQLite with FTS5 for search
- MCP protocol (stdio transport)

File picker implemented with:
- Backend: internal/app/picker.go
- Frontend: web/src/path-picker.tsx
- Native dialog via System Events on macOS

**Evidence**: Session f75cb010, messages 15-42
```

**Updated**: As agent learns new facts  
**Used by**: Providing technical context

### 4. Summary

Per-session summaries capturing what was done.

```markdown
<!-- memories/summary/2026-10-15-redesign.md -->
# UI Redesign Session

**Goal**: Redesign frontend from scratch

**Completed**:
- Created design system (4-tier radius, token-based spacing)
- Implemented folder picker with native dialog fallback
- Added path browser component
- Updated all page styles

**Decisions**:
- Use System Events for macOS native dialogs
- Fall back to in-app browser when native unavailable
- Unified border radius: 4/6/10/14px tiers

**Next**: Test picker on Windows and Linux

**Sessions**: f75cb010b12a8eb0...
```

**Updated**: At session end  
**Used by**: Resuming work later

## Agent Workflow

### Session Start

```
1. Agent launches
       ↓
2. SessionStart hook → creates session
       ↓
3. Agent calls memvault_context
       ↓
4. Returns: background + preferences + relevant history
       ↓
5. Agent has full context before first message
```

Example MCP call:

```json
{
  "method": "tools/call",
  "params": {
    "name": "memvault_context",
    "arguments": {
      "session": "current",
      "query": "folder picker"  // Optional: focus on specific topic
    }
  }
}
```

Response:

```json
{
  "core": [
    {
      "kind": "background",
      "title": "User Identity",
      "body": "Software engineer..."
    },
    {
      "kind": "preference",
      "title": "Code Style",
      "body": "Prefer functional patterns..."
    }
  ],
  "related": [
    {
      "session": "abc123",
      "title": "Previous picker work",
      "relevance": 0.85
    }
  ]
}
```

### During Conversation

Agent can search memories and archives:

```json
{
  "name": "memvault_search",
  "arguments": {
    "query": "how did we implement X?",
    "kind": "memory",  // or "session"
    "limit": 5
  }
}
```

### Learning New Facts

When agent discovers preferences or knowledge:

```json
{
  "name": "memvault_memory_update",
  "arguments": {
    "kind": "preference",
    "title": "Error Handling Style",
    "body": "User prefers wrapping errors with context in Go using fmt.Errorf with %w",
    "evidence": [
      {
        "session": "current",
        "message": "msg_015"
      }
    ]
  }
}
```

### Session End

Agent writes summary:

```json
{
  "name": "memvault_memory_update",
  "arguments": {
    "kind": "summary",
    "session": "current",
    "title": "Folder Picker Implementation",
    "body": "**Goal**: Add native folder picker...\n**Completed**: ...\n**Next**: ...",
    "covered": ["msg_001", "msg_002", "..."]
  }
}
```

## Configuration

### Enable MCP Integration

Edit `~/.claude/settings.json`:

```json
{
  "mcp": {
    "servers": {
      "memvault": {
        "command": "memvault",
        "args": ["mcp"],
        "env": {
          "MEMVAULT_HOME": "/Users/you/memvault"
        }
      }
    }
  }
}
```

Restart Claude Code to load the server.

### Verify Integration

```bash
# In a Claude Code session, the agent should have access to:
# - memvault_context
# - memvault_search
# - memvault_memory_update
# - memvault_session_read
# - memvault_session_link
# - memvault_tasks

# Check manually:
memvault mcp --check
```

## Memory Maintenance

### Automatic Maintenance

Agents automatically:
- Extract facts from conversations
- Update existing memories
- Create summaries at session end
- Link related sessions

Based on [memory management patterns](https://github.com/scanadi/mcp-ai-memory/), successful systems balance automatic extraction with human oversight.

### Manual Review

List memories:

```bash
# All memories
memvault memories list

# By type
memvault memories list --kind preference

# Recently updated
memvault memories list --sort updated
```

Read memory:

```bash
memvault memories read <memory-id>
```

Edit memory directly:

```bash
# Memories are Markdown files
vim ~/memvault/memories/preference/code-style.md
```

### Conflict Resolution

When memories are edited by multiple agents or on multiple machines:

```markdown
<!-- memories/preference/language.md -->
<<<<<<< HEAD
User prefers Go for system tools
=======
User prefers Rust for system tools
>>>>>>> Synced from laptop
```

Resolve manually:

```bash
# List conflicts
memvault memories conflicts

# View conflicted memory
memvault memories read <id>

# Edit to resolve
vim ~/memvault/memories/preference/language.md

# Mark as resolved
memvault memories resolve <id>
```

## Best Practices

### 1. Let Agents Maintain Memories

**Don't**: Manually write every preference

**Do**: Let agents discover and record preferences through natural conversation

```
User: "I prefer short variable names"
Agent: (records preference automatically)
```

### 2. Provide Evidence

Memories should always link back to source:

```markdown
**Evidence**: Session abc123, message 42
```

This enables:
- Verification
- Context recovery
- Conflict resolution

### 3. Keep Memories Focused

**Bad** (too broad):
```markdown
# Everything About User
User is a developer who likes Go and prefers...
(5000 words)
```

**Good** (focused):
```markdown
# Language Preference for CLIs
User prefers Go for command-line tools.

**Why**: Single-binary distribution, fast startup.
```

### 4. Update, Don't Duplicate

When learning new information:

**Don't**: Create "Language Preference 2"

**Do**: Update existing "Language Preference" memory

Agents use the `parents` field to track revisions:

```markdown
<!-- parents: abc123..., def456... -->
```

### 5. Use Appropriate Memory Types

- **Background**: Stable facts about user
- **Preference**: How user likes things done
- **Knowledge**: Technical facts about projects
- **Summary**: What happened in a session

Don't put session-specific details in background or permanent preferences in summaries.

## Advanced Usage

### Cross-Agent Context

Agents can access each other's memories:

```json
{
  "name": "memvault_context",
  "arguments": {
    "agent": "codex",  // Read Codex's memories
    "query": "how did codex solve X?"
  }
}
```

This enables:
- Learning from other agents
- Consistent behavior across agents
- Knowledge transfer

### Session Linking

Track conversation relationships:

```bash
# This session continues previous work
memvault sessions link \
  --from old-session \
  --to new-session \
  --type continuation

# Agent can then follow the chain
memvault sessions read old-session
```

### Memory Search

Agents can search memories semantically:

```json
{
  "name": "memvault_search",
  "arguments": {
    "query": "user's testing preferences",
    "kind": "memory",
    "limit": 3
  }
}
```

Returns relevant memories ranked by similarity.

### Task-Based Updates

Query pending maintenance:

```json
{
  "name": "memvault_tasks",
  "arguments": {
    "op": "list"
  }
}
```

Returns:
```json
{
  "tasks": [
    {
      "id": "summarize-abc123",
      "kind": "summarize",
      "session": "abc123",
      "pending": 15  // 15 messages not yet summarized
    }
  ]
}
```

Claim and complete:

```json
{
  "name": "memvault_tasks",
  "arguments": {
    "op": "claim",
    "id": "summarize-abc123",
    "owner": "claude"
  }
}
```

## Troubleshooting

### Agent Not Getting Context

**Symptom**: Agent acts like it's the first time

**Check**:

```bash
# 1. Is MCP server configured?
cat ~/.claude/settings.json | grep memvault

# 2. Is server running?
ps aux | grep "memvault mcp"

# 3. Do memories exist?
memvault memories list

# 4. Check logs
tail -f ~/.memvault/logs/mcp.log
```

### Memories Not Updating

**Symptom**: Agent learning same things repeatedly

**Causes**:
- Agent not calling `memvault_memory_update`
- Permission errors
- Conflicts blocking writes

**Solutions**:

```bash
# Check recent updates
memvault memories list --sort updated

# Check for conflicts
memvault memories conflicts

# Check permissions
ls -la ~/memvault/memories/
```

### Memory Conflicts

**Symptom**: Conflicting information in memories

**Resolve**:

```bash
# List conflicts
memvault memories conflicts

# Read conflicted memory
memvault memories read <id>

# Edit to merge
vim ~/memvault/memories/path/to/memory.md

# Remove conflict markers
# Keep correct information
# Add note about why it changed
```

### Old Context Retrieved

**Symptom**: Agent using outdated information

**Causes**:
- Stale memories not updated
- Search index not rebuilt
- Agent reading old linked sessions

**Solutions**:

```bash
# Rebuild search index
memvault index rebuild

# Check memory update dates
memvault memories list --sort updated

# Update outdated memory manually
vim ~/memvault/memories/knowledge/project-info.md
```

## Performance

### Context Retrieval Speed

| Operation | Typical Time |
|-----------|--------------|
| `memvault_context` | 10-50ms |
| `memvault_search` (memories) | 5-20ms |
| `memvault_memory_update` | 5-20ms |
| `memvault_session_read` | 10-50ms |

These benchmarks align with [production MCP implementations](https://github.com/adamrdrew/agent-memory-mcp) optimized for sub-100ms response times.

### Memory Size Management

```bash
# Check memory usage
du -sh ~/memvault/memories/

# Archive old summaries
mv ~/memvault/memories/summary/2025-*.md \
   ~/memvault-archive/summaries/

# Consolidate similar memories
# (manually merge overlapping preferences)
```

## Sources

- [Oracle: Context Assembly for Multi-Tenant Agent Memory](https://blogs.oracle.com/developers/context-assembly-for-multi-tenant-agent-memory-budgets-selection-and-tool-calls-with-mcp)
- [MCP Memory Service Implementation](https://github.com/doobidoo/mcp-memory-service/wiki/)
- [MCP AI Memory: Production-Ready Semantic Memory](https://github.com/scanadi/mcp-ai-memory/)
- [Building AI Agent Knowledge Base with MCP](https://growwstacks.com/blog/building-ai-agent-knowledge-base-mcp-rag-vector-db)
- [Agent Memory MCP: Hybrid BM25 + Vector Search](https://github.com/adamrdrew/agent-memory-mcp)

## Next Steps

- **[MCP Integration Concepts](/concepts/mcp)** - Protocol details
- **[Search Guide](/guide/search)** - Query memories and archives
- **[CLI Reference](/reference/cli)** - Memory management commands

---

**Last reviewed**: October 2026. [Contribute improvements](https://github.com/liyown/memvault/blob/main/CONTRIBUTING.md).
