# Conversation vs Memory

Understanding Memvault's dual-layer architecture.

## The Two Layers

### 1. Conversations (Archive Layer)

**Complete, immutable records** of all agent interactions.

- Stored in structured format (JSON + Markdown)
- One file per session
- Never modified after creation
- Contains:
  - Full message history
  - Attachments and artifacts
  - Metadata (timestamps, agent, model)
  - Edit history

**Think of it as**: Your conversation database.

**Use for**:
- Historical reference
- Full-text search
- Audit trail
- Data export

### 2. Memories (Semantic Layer)

**Extracted, agent-maintained knowledge** from conversations.

- Stored as Markdown files
- Continuously updated
- Four types:
  - **Background**: Who you are, role, expertise
  - **Preference**: How you like to work
  - **Knowledge**: Facts learned from conversations
  - **Summary**: Session summaries

**Think of it as**: Your agent's working memory.

**Use for**:
- Providing context to agents
- Cross-session continuity
- Personalization
- Knowledge retention

## How They Interact

```
┌─────────────────┐
│  Agent Session  │
└────────┬────────┘
         │
         ├─────► Archives (complete record)
         │       └─► Full-text index
         │
         └─────► Memories (extracted facts)
                 └─► Agent context
```

### During Conversation

1. Agent asks for memory context via MCP
2. Memvault returns relevant memories
3. Conversation happens
4. Raw transcript saved to archive
5. Agent updates memories based on new learnings

### Between Sessions

- Archives remain unchanged
- Memories can be:
  - Read by new agent sessions
  - Updated with new facts
  - Conflicted (when edited by multiple agents)
  - Exported for sync

## Examples

### Conversation Record

```json
// sessions/f75cb010/messages/0001.json
{
  "role": "user",
  "content": "I prefer Go for system tools",
  "timestamp": "2026-10-15T08:00:00Z"
}
```

### Extracted Memory

```markdown
<!-- memories/preference/language-choice.md -->
# Language Preference for System Tools

User prefers Go for building system utilities and CLI tools.

**Why**: Performance, single-binary distribution, cross-platform.

**Evidence**: session f75cb010, message 0001
```

## Key Differences

| Aspect | Conversations | Memories |
|--------|--------------|----------|
| **Mutability** | Immutable | Mutable |
| **Granularity** | Message-level | Fact-level |
| **Structure** | Chronological | Semantic |
| **Size** | Large (full text) | Small (key facts) |
| **Access** | Search | Direct load |
| **Lifetime** | Permanent | Updated |

## Agent Workflow

### Reading Memory

Before answering:

```
Agent: memvault_context(session="current")
→ Returns: background, preferences, relevant knowledge
Agent: Uses context to inform response
```

### Writing Memory

After important exchange:

```
Agent: memvault_memory_update(
  kind="preference",
  title="Code Review Depth",
  body="User wants security-focused reviews",
  evidence=[{session, message}]
)
→ Memory saved as Markdown
```

## Why Both?

### Archives Alone

❌ No semantic understanding  
❌ Agent must re-read everything  
❌ Slow context retrieval  

### Memories Alone

❌ Lose original context  
❌ No audit trail  
❌ Can't correct mistakes  

### Together

✅ Fast semantic access (memories)  
✅ Complete audit trail (archives)  
✅ Agents build context efficiently  
✅ Human can verify and correct  

## Memory Types in Detail

### Background

Who you are:

```markdown
# User Identity

Software engineer focused on distributed systems.
Works with Go, Rust, and TypeScript.
```

### Preference

How you work:

```markdown
# Code Style

- Prefers functional over imperative
- Always uses error wrapping in Go
- Wants tests before implementation
```

### Knowledge

Facts learned:

```markdown
# Project Architecture

memvault uses SQLite FTS5 for full-text search.
Index built from message.content and message.meta.
```

### Summary

Session recap:

```markdown
# Session Summary

**Goal**: Implement folder picker UI  
**Completed**: Added native dialog fallback  
**Next**: Test on Windows  
```

## Managing Both Layers

### View Archives

```bash
# List sessions
memvault sessions list

# Read session
memvault sessions read <id>

# Search across all
memvault search "query"
```

### View Memories

```bash
# List memories
memvault memories list

# Read memory
memvault memories read <id>

# Search memories
memvault memories search "query"
```

### Export for Sync

```bash
# Export both
memvault export --to /path/to/sync

# Import on another machine
memvault import --from /path/to/sync
```

## Best Practices

### For Archives

- ✅ Never manually edit
- ✅ Use search for retrieval
- ✅ Export regularly for backup
- ✅ Let hooks capture automatically

### For Memories

- ✅ Let agents maintain them
- ✅ Review for accuracy
- ✅ Edit Markdown directly when needed
- ✅ Keep evidence links intact

## Next Steps

- [Why Local-First?](/concepts/why-local-first)
- [Storage Design](/concepts/storage)
- [Agent Memory Guide](/guide/memory)
