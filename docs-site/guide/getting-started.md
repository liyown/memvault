# Getting Started

Get Memvault running in 5 minutes.

## Prerequisites

- Go 1.23 or later
- macOS, Linux, or Windows
- Supported agent: Claude Code or Codex

## Installation

### Option 1: From Source

```bash
git clone https://github.com/liyown/memvault.git
cd memvault
go build -o bin/memvault ./cmd/memvault
```

Add to PATH:
```bash
# macOS/Linux
export PATH="$PATH:$(pwd)/bin"

# Or copy to a directory in PATH
cp bin/memvault /usr/local/bin/
```

### Option 2: Direct Install

```bash
go install github.com/liyown/memvault/cmd/memvault@latest
```

## Verify Installation

```bash
memvault version
```

## Initial Setup

### 1. Initialize Vault

```bash
memvault init
```

This creates:
- `~/memvault` - conversation archives
- `~/.memvault/config.yaml` - configuration
- Database indices for search

### 2. Configure Agent Hooks

#### Claude Code

Edit `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": "memvault hook session-start",
    "UserPromptSubmit": "memvault hook user-prompt-submit"
  }
}
```

Restart Claude Code to activate hooks.

#### Codex

Edit Codex settings:

```json
{
  "hooks": {
    "onSessionStart": "memvault hook session-start",
    "onUserPrompt": "memvault hook user-prompt-submit"
  }
}
```

### 3. Start Server (Optional)

For web UI and MCP server:

```bash
memvault serve
```

Default: `http://localhost:27961`

## Verify It Works

### Archive Check

Have a conversation in your agent, then:

```bash
memvault sessions list
```

You should see your session listed.

### Search Check

```bash
memvault search "your query"
```

### MCP Check

In Claude Code, the agent should now have access to `memvault_*` tools.

## Next Steps

- [Archiving Guide](/guide/archiving) - How conversation capture works
- [Search Guide](/guide/search) - Full-text search usage
- [Memory Guide](/guide/memory) - Agent memory system
- [CLI Reference](/reference/cli) - All commands

## Troubleshooting

### "Hooks not firing"

1. Check hook configuration in agent settings
2. Restart agent after adding hooks
3. Verify `memvault` is in PATH: `which memvault`

### "No sessions captured"

1. Ensure hooks are configured (see above)
2. Check agent is supported (Claude Code / Codex)
3. Look for errors: `tail -f ~/.memvault/logs/memvault.log`

### "MCP tools not available"

1. Ensure server is running: `memvault serve`
2. Check MCP config: `cat ~/.config/claude/mcp.json`
3. Restart Claude Code

For more help, see [Configuration Reference](/reference/configuration).
