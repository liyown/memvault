# MCP Tools Reference

Complete reference for Memvault's Model Context Protocol (MCP) tools available to AI agents.

## Overview

Memvault exposes these tools via MCP:
- Session capture and retrieval
- Memory management
- Full-text search
- Session linking
- Task management

All tools follow the [MCP specification](https://spec.modelcontextprotocol.io/) using stdio transport.

## Configuration

Add to agent settings (e.g., `~/.claude/settings.json`):

```json
{
  "mcp": {
    "servers": {
      "memvault": {
        "command": "memvault",
        "args": ["mcp"],
        "env": {
          "MEMVAULT_HOME": "/path/to/vault"
        }
      }
    }
  }
}
```

Verify:

```bash
memvault mcp --check
```

## Tools

### `memvault_context`

Retrieve contextual information for the current agent: background, preferences, and relevant history.

**Parameters**:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `agent` | string | No | Agent name (default: current) |
| `native_id` | string | No | Session ID |
| `query` | string | No | Focus context on specific topic |
| `session` | string | No | Session identifier |

**Returns**:

```json
{
  "core": [
    {
      "id": "abc123...",
      "title": "User Identity",
      "kind": "background",
      "body": "Software engineer...",
      "evidence": [...],
      "updated": "2026-10-15T10:00:00Z",
      "author": "claude"
    }
  ],
  "related": [
    {
      "session": "def456",
      "title": "Previous work on X",
      "relevance": 0.85,
      "timestamp": "2026-10-10T14:30:00Z"
    }
  ]
}
```

**Example**:

```json
{
  "method": "tools/call",
  "params": {
    "name": "memvault_context",
    "arguments": {
      "query": "folder picker implementation"
    }
  }
}
```

**Use Cases**:
- Session start: Load user background and preferences
- Topic-specific context: Focus on relevant memories
- Cross-agent learning: Read another agent's context

### `memvault_memory_update`

Create or update a semantic memory.

**Parameters**:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `kind` | enum | Yes | `background`, `preference`, `knowledge`, `summary` |
| `title` | string | Yes | Memory title |
| `body` | string | Yes | Memory content (Markdown) |
| `id` | string | No | Update existing memory |
| `parents` | array | No | Parent memory versions |
| `evidence` | array | No | Source messages |
| `covered` | array | No | Message versions covered |
| `session` | string | No | Session identifier |
| `author` | string | No | Author name |
| `pending` | boolean | No | Mark as pending review |

**Evidence format**:

```json
{
  "session": "abc123",
  "message": "msg_042"
}
```

**Returns**:

```json
{
  "id": "memory-abc123...",
  "revision": "revision-hash",
  "created": true
}
```

**Example - Create preference**:

```json
{
  "name": "memvault_memory_update",
  "arguments": {
    "kind": "preference",
    "title": "Error Handling Style",
    "body": "User prefers wrapping errors with context using fmt.Errorf with %w in Go",
    "evidence": [
      {
        "session": "current",
        "message": "msg_015"
      }
    ]
  }
}
```

**Example - Update existing**:

```json
{
  "name": "memvault_memory_update",
  "arguments": {
    "id": "existing-memory-id",
    "parents": ["parent-revision-hash"],
    "title": "Updated Preferences",
    "body": "Updated content...",
    "kind": "preference"
  }
}
```

**Use Cases**:
- Record user preferences discovered during conversation
- Update project knowledge as it evolves
- Create session summaries at end
- Track decision rationale

### `memvault_search`

Full-text search across sessions and memories.

**Parameters**:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Search query (FTS5 syntax) |
| `agent` | string | No | Filter by agent |
| `kind` | enum | No | `session`, `memory`, or omit for all |
| `role` | enum | No | `user`, `assistant` |
| `limit` | integer | No | Max results (default: 10) |
| `offset` | integer | No | Pagination offset |
| `sort` | enum | No | `relevance` (default), `recent` |

**Returns**:

```json
{
  "results": [
    {
      "session": "abc123",
      "message": "msg_042",
      "score": 0.95,
      "snippet": "...implement folder **picker** with native...",
      "timestamp": "2026-10-15T08:23:45Z",
      "role": "user"
    }
  ],
  "total": 42,
  "query_time_ms": 15
}
```

**Examples**:

Simple search:
```json
{
  "name": "memvault_search",
  "arguments": {
    "query": "folder picker"
  }
}
```

Search memories only:
```json
{
  "name": "memvault_search",
  "arguments": {
    "query": "user preferences",
    "kind": "memory",
    "limit": 5
  }
}
```

Complex query:
```json
{
  "name": "memvault_search",
  "arguments": {
    "query": "implementation AND (picker OR dialog)",
    "agent": "claude",
    "sort": "recent"
  }
}
```

**Query Syntax**: See [Search Guide](/guide/search) for full FTS5 syntax.

**Use Cases**:
- Find how similar problems were solved
- Retrieve relevant conversation history
- Search user preferences on specific topics
- Discover related discussions

### `memvault_session_read`

Read complete session transcript.

**Parameters**:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `session` | string | Yes | Session ID |
| `limit` | integer | No | Max messages |
| `offset` | integer | No | Skip messages |
| `message` | string | No | Start from specific message |

**Returns**:

```json
{
  "session": {
    "id": "abc123",
    "agent": "claude",
    "native_id": "4b67a8e4...",
    "title": "Folder Picker Implementation",
    "created": "2026-10-15T08:00:00Z",
    "updated": "2026-10-15T10:30:00Z",
    "message_count": 42
  },
  "messages": [
    {
      "version": "1.0",
      "role": "user",
      "content": "Implement folder picker",
      "timestamp": "2026-10-15T08:23:45Z"
    },
    {
      "role": "assistant",
      "content": "I'll implement...",
      "timestamp": "2026-10-15T08:24:12Z"
    }
  ]
}
```

**Example**:

```json
{
  "name": "memvault_session_read",
  "arguments": {
    "session": "abc123",
    "limit": 50
  }
}
```

**Use Cases**:
- Resume work from previous session
- Review full context of linked sessions
- Extract complete conversation for analysis

### `memvault_session_link`

Create relationship between sessions.

**Parameters**:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `from` | string | Yes | Source session ID |
| `to` | string | Yes | Target session ID |
| `kind` | enum | Yes | `continuation`, `branch`, `reference` |

**Link Types**:

- **continuation**: Direct continuation of work
- **branch**: Alternative approach from same starting point
- **reference**: Related but independent discussion

**Returns**:

```json
{
  "link_id": "link-abc123",
  "from": "session-1",
  "to": "session-2",
  "kind": "continuation",
  "created": "2026-10-15T10:00:00Z"
}
```

**Example**:

```json
{
  "name": "memvault_session_link",
  "arguments": {
    "from": "old-session-id",
    "to": "new-session-id",
    "kind": "continuation"
  }
}
```

**Use Cases**:
- Track project evolution across sessions
- Connect related discussions
- Enable context navigation

### `memvault_tasks`

Query or claim maintenance tasks.

**Parameters**:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `op` | enum | Yes | `list`, `claim` |
| `id` | string | No | Task ID (for claim) |
| `owner` | string | No | Claim owner |

**Returns (list)**:

```json
{
  "tasks": [
    {
      "id": "summarize-abc123",
      "kind": "summarize",
      "session": "abc123",
      "pending": 15,
      "priority": "normal",
      "created": "2026-10-15T10:00:00Z"
    }
  ]
}
```

**Returns (claim)**:

```json
{
  "claimed": true,
  "task_id": "summarize-abc123",
  "expires_at": "2026-10-15T10:05:00Z"
}
```

**Example - List tasks**:

```json
{
  "name": "memvault_tasks",
  "arguments": {
    "op": "list"
  }
}
```

**Example - Claim task**:

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

**Task Types**:

- **summarize**: Create session summary
- **consolidate**: Merge overlapping memories
- **index**: Update search index
- **cleanup**: Remove temporary files

**Use Cases**:
- Background memory maintenance
- Distributed work across agents
- Deferred expensive operations

## Error Handling

All tools return errors in standard MCP format:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Session not found: abc123"
  }
}
```

**Common Error Codes**:

| Code | Description |
|------|-------------|
| `INVALID_PARAMS` | Missing or invalid parameters |
| `NOT_FOUND` | Resource not found |
| `CONFLICT` | Concurrent modification |
| `PERMISSION_DENIED` | Insufficient permissions |
| `INTERNAL_ERROR` | Server error |

## Performance

Typical response times:

| Tool | Typical Time |
|------|--------------|
| `memvault_context` | 10-50ms |
| `memvault_search` | 5-100ms |
| `memvault_memory_update` | 5-20ms |
| `memvault_session_read` | 10-50ms |
| `memvault_session_link` | 5-10ms |
| `memvault_tasks` | 5-10ms |

Performance scales with:
- Vault size
- Query complexity
- Disk speed

## Batching

MCP supports batching multiple tool calls:

```json
{
  "method": "tools/call",
  "params": {
    "batch": [
      {
        "name": "memvault_context",
        "arguments": {"query": "project background"}
      },
      {
        "name": "memvault_search",
        "arguments": {"query": "recent bugs", "limit": 5}
      }
    ]
  }
}
```

Memvault executes batches in parallel where possible.

## Security

### Data Privacy

- All data stays local
- No network calls to Anthropic or third parties
- MCP communication via stdio only

### Access Control

- Agents have full access to vault
- No per-tool permissions currently
- Trust agent implementation

### Audit Trail

All tool calls logged:

```bash
tail -f ~/.memvault/logs/mcp.log
```

Example log:

```
2026-10-15T10:00:00Z INFO memvault_search query="folder picker" agent=claude results=5 duration=15ms
2026-10-15T10:00:15Z INFO memvault_memory_update kind=preference title="Code Style" agent=claude
```

## Testing

Test MCP integration:

```bash
# Start MCP server
memvault mcp

# In another terminal, send test request
echo '{"method":"tools/list","params":{}}' | memvault mcp
```

Manual tool invocation:

```bash
# Test search
echo '{
  "method": "tools/call",
  "params": {
    "name": "memvault_search",
    "arguments": {"query": "test"}
  }
}' | memvault mcp
```

## Debugging

Enable debug logging:

```bash
MEMVAULT_LOG_LEVEL=debug memvault mcp
```

View detailed logs:

```bash
tail -f ~/.memvault/logs/mcp.log
```

Common issues:

**Tool not found**:
- Verify `memvault mcp --check`
- Restart agent to reload MCP config

**No results from context**:
- Check memories exist: `memvault memories list`
- Verify session captured: `memvault sessions list`

**Slow responses**:
- Rebuild index: `memvault index rebuild`
- Check disk speed: `iostat`

## Examples

### Agent Workflow

Complete agent session workflow:

```python
# 1. Session start
context = mcp_call("memvault_context", {
    "query": "current project"
})

# 2. During conversation - search for relevant info
results = mcp_call("memvault_search", {
    "query": "previous implementation",
    "limit": 3
})

# 3. Learn preference
mcp_call("memvault_memory_update", {
    "kind": "preference",
    "title": "Testing Strategy",
    "body": "User prefers table-driven tests in Go",
    "evidence": [{"session": "current", "message": "msg_042"}]
})

# 4. Session end - create summary
mcp_call("memvault_memory_update", {
    "kind": "summary",
    "session": "current",
    "title": "Implemented Feature X",
    "body": "**Goal**: ...\n**Completed**: ...\n**Next**: ...",
    "covered": ["msg_001", "msg_002", "..."]
})
```

## Next Steps

- **[Agent Memory Guide](/guide/memory)** - Memory system usage
- **[MCP Concepts](/concepts/mcp)** - Protocol overview
- **[CLI Reference](/reference/cli)** - Command-line tools

## Sources

- [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/)
- [MCP SDK Documentation](https://github.com/anthropics/mcp-sdk)
- [Oracle: Multi-Tenant Agent Memory with MCP](https://blogs.oracle.com/developers/context-assembly-for-multi-tenant-agent-memory-budgets-selection-and-tool-calls-with-mcp)

---

**Last updated**: October 2026
