# MCP Integration

How Memvault integrates with the **Model Context Protocol** to give agents memory.

## What is MCP?

The **Model Context Protocol** is Anthropic's standard for connecting AI agents to external data sources.

With MCP, agents can:
- Read data from local tools
- Write data to local services
- Access resources securely
- Call functions with type safety

## Memvault as an MCP Server

Memvault exposes its functionality through MCP tools:

```
┌──────────────────────────────────┐
│      Claude Code / Agent         │
└──────────────┬───────────────────┘
               │ MCP Protocol
┌──────────────▼───────────────────┐
│   Memvault MCP Server (stdio)    │
├──────────────────────────────────┤
│  • memvault_context             │
│  • memvault_search              │
│  • memvault_memory_update       │
│  • memvault_session_read        │
│  • ... (more tools)             │
└──────────────┬───────────────────┘
               │
┌──────────────▼───────────────────┐
│  Local Vault (files + SQLite)    │
└──────────────────────────────────┘
```

## Available MCP Tools

### Core Tools

#### `memvault_context`
Get session state, memories, and history.

```json
{
  "name": "memvault_context",
  "arguments": {
    "session": "current",
    "query": "optional search term"
  }
}
```

Returns:
- Core memories (background, preferences)
- Session summary
- Related historical context
- Pending messages

#### `memvault_search`
Full-text search across all sessions and memories.

```json
{
  "name": "memvault_search",
  "arguments": {
    "query": "folder picker",
    "limit": 10,
    "kind": "session" // or "memory"
  }
}
```

#### `memvault_memory_update`
Create or update agent memory.

```json
{
  "name": "memvault_memory_update",
  "arguments": {
    "kind": "preference",
    "title": "Code Review Style",
    "body": "User wants security-focused reviews",
    "evidence": [
      {"session": "abc123", "message": "msg_01"}
    ]
  }
}
```

#### `memvault_session_read`
Read full session messages with pagination.

```json
{
  "name": "memvault_session_read",
  "arguments": {
    "session": "abc123",
    "offset": 0,
    "limit": 100
  }
}
```

### Link Management

#### `memvault_session_link`
Create relationships between sessions.

```json
{
  "name": "memvault_session_link",
  "arguments": {
    "from": "session_a",
    "to": "session_b",
    "kind": "continuation" // or "branch", "reference"
  }
}
```

### Maintenance

#### `memvault_tasks`
Query and claim maintenance tasks.

```json
{
  "name": "memvault_tasks",
  "arguments": {
    "op": "list" // or "claim"
  }
}
```

## Setup

### 1. Install Memvault

```bash
go install github.com/liyown/memvault/cmd/memvault@latest
```

### 2. Start MCP Server

#### Claude Code

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

#### Codex

Similar MCP configuration in Codex settings.

### 3. Restart Agent

Quit and restart Claude Code or Codex to load the MCP server.

## How It Works

### stdio Transport

Memvault uses **stdio** (standard input/output) for MCP:

```
Agent Process          Memvault Process
     │                       │
     ├──────request──────────>│
     │   (JSON-RPC)           │
     │                        │
     │<─────response──────────┤
     │   (JSON-RPC)           │
```

- Fast: No network overhead
- Secure: No exposed ports
- Simple: No authentication needed

### Session Identity

Hook provides session identity:

```json
// Via hook at session start
{
  "agent": "claude",
  "native_session_id": "4b67a8e4-b697-4ede-..."
}
```

Memvault maps to internal session ID:

```json
// Internal session
{
  "id": "f75cb010b12a8eb0...",
  "native_id": "4b67a8e4-b697-4ede-...",
  "agent": "claude"
}
```

### Context Injection

At session start:

1. Hook fires → creates session
2. Agent calls `memvault_context`
3. Memvault returns memories
4. Injected into agent's context

```
Agent Start
   ↓
Hook: session-start
   ↓
MCP: memvault_context
   ↓
Returns: background, preferences, summaries
   ↓
Agent has full context
```

## Agent Workflow

### Reading Context

```typescript
// At start of session
const context = await mcp.call("memvault_context", {
  session: "current"
});

// Context contains:
// - core memories (who user is, preferences)
// - session summary (what we were doing)
// - related history (similar past conversations)
```

### Searching History

```typescript
// When user asks about past work
const results = await mcp.call("memvault_search", {
  query: "folder picker implementation",
  limit: 5
});

// Returns relevant sessions and memories
```

### Updating Memory

```typescript
// After important learning
await mcp.call("memvault_memory_update", {
  kind: "knowledge",
  title: "Project uses Go",
  body: "This codebase is Go-based. Uses embed for assets.",
  evidence: [{
    session: currentSession,
    message: messageId
  }]
});
```

## Protocol Details

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "memvault_search",
    "arguments": {
      "query": "test"
    }
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{ results JSON }"
      }
    ]
  }
}
```

### Error Handling

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32603,
    "message": "Session not found"
  }
}
```

## Security Model

### No Authentication

- stdio transport → same user
- No network exposure
- Process isolation via OS

### Permission Model

- Read: Always allowed
- Write: Requires session ownership
- Delete: Not exposed via MCP

### Data Validation

- All inputs validated
- SQL injection prevented (prepared statements)
- Path traversal blocked
- Size limits enforced

## Debugging

### Enable Verbose Mode

```bash
export MEMVAULT_LOG_LEVEL=debug
memvault mcp
```

### Test Tool Manually

```bash
# Send JSON-RPC request
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memvault_search","arguments":{"query":"test"}}}' | memvault mcp
```

### Check Logs

```bash
tail -f ~/.memvault/logs/mcp.log
```

## Performance

### Latency

| Operation | Typical Time |
|-----------|--------------|
| `memvault_context` | 10-50ms |
| `memvault_search` | 20-100ms |
| `memvault_memory_update` | 5-20ms |
| `memvault_session_read` | 10-50ms |

### Caching

- Session context cached
- Search index in memory
- File reads cached by OS

## Comparison with Other Approaches

### Webhooks

❌ Requires server  
❌ Network latency  
❌ Authentication needed  
❌ Exposed to network  

### REST API

❌ Port binding  
❌ HTTP overhead  
❌ CORS issues  
❌ Auth complexity  

### MCP (stdio)

✅ No server needed  
✅ Zero network overhead  
✅ OS-level security  
✅ Simple integration  

## Future Extensions

### Planned Tools

- `memvault_export` - Export sessions
- `memvault_stats` - Usage statistics
- `memvault_conflicts` - List memory conflicts

### Resource Providers

MCP resources for:
- Memory files (read via URI)
- Session archives (read via URI)
- Search indices (query via URI)

## Next Steps

- [Agent Memory Guide](/guide/memory) - Using memory in practice
- [CLI Reference](/reference/cli) - Command-line tools
- [Configuration](/reference/configuration) - MCP settings
