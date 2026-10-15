# Installation

## System Requirements

- **Operating System**: macOS, Linux, or Windows
- **Go Version**: 1.23 or later
- **Disk Space**: ~100MB for binary, plus storage for archives
- **Supported Agents**:
  - Claude Code (terminal and desktop)
  - Codex by Anthropic
  - OpenCode (coming soon)

## Installation Methods

### 1. From Source (Recommended)

Clone and build:

```bash
git clone https://github.com/liyown/memvault.git
cd memvault
go build -o bin/memvault ./cmd/memvault
```

Install to system:

```bash
# macOS/Linux
sudo cp bin/memvault /usr/local/bin/

# Or add to PATH in your shell rc file
echo 'export PATH="$PATH:'$(pwd)'/bin"' >> ~/.zshrc
source ~/.zshrc
```

```powershell
# Windows (PowerShell as Administrator)
Copy-Item bin\memvault.exe C:\Windows\System32\
```

### 2. Go Install

```bash
go install github.com/liyown/memvault/cmd/memvault@latest
```

Ensure `$GOPATH/bin` is in your PATH:

```bash
# macOS/Linux
export PATH="$PATH:$(go env GOPATH)/bin"

# Add to ~/.zshrc or ~/.bashrc to persist
```

### 3. Pre-built Binaries

Download from [GitHub Releases](https://github.com/liyown/memvault/releases):

```bash
# macOS (ARM)
curl -L https://github.com/liyown/memvault/releases/latest/download/memvault-darwin-arm64.tar.gz | tar xz
sudo mv memvault /usr/local/bin/

# Linux (AMD64)
curl -L https://github.com/liyown/memvault/releases/latest/download/memvault-linux-amd64.tar.gz | tar xz
sudo mv memvault /usr/local/bin/
```

## Verify Installation

```bash
memvault version
# Output: memvault version v0.1.0
```

```bash
memvault help
# Shows available commands
```

## Initial Configuration

### 1. Initialize Vault

```bash
memvault init
```

This creates:

```
~/memvault/               # Default vault directory
├── sessions/             # Conversation archives
├── memories/             # Agent memory files
└── .memvault/
    ├── db/               # Search indices
    └── config.yaml       # Configuration
```

### 2. Custom Vault Location

```bash
export MEMVAULT_HOME=/path/to/vault
memvault init
```

Or set in config:

```yaml
# ~/.memvault/config.yaml
vault:
  root: /path/to/vault
```

### 3. Agent Home (for hooks)

For multi-user systems or agent-specific vaults:

```bash
export MEMVAULT_AGENT_HOME=/path/to/agent/vault
```

## Agent Integration

### Claude Code

Edit `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": "memvault hook session-start",
    "UserPromptSubmit": "memvault hook user-prompt-submit"
  },
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

### Codex

Edit Codex hook configuration:

```json
{
  "hooks": {
    "onSessionStart": "memvault hook session-start --agent codex",
    "onUserPrompt": "memvault hook user-prompt-submit --agent codex"
  }
}
```

### Restart Agents

After adding hooks:
1. **Claude Code**: Quit and reopen terminal/desktop app
2. **Codex**: Restart Codex application

## Start Server

For web UI and MCP services:

```bash
# Foreground
memvault serve

# Background (macOS/Linux)
nohup memvault serve > /tmp/memvault.log 2>&1 &

# As system service (systemd)
sudo systemctl start memvault
```

Default port: `27961`

Web UI: `http://localhost:27961`

## Verification

### Check Archive

Have a conversation, then:

```bash
memvault sessions list
```

### Check Search

```bash
memvault search "test"
```

### Check MCP

In Claude Code, the agent should list `memvault_*` tools.

## Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/memvault

# Remove data (optional)
rm -rf ~/memvault ~/.memvault
```

## Next Steps

- [Getting Started](/guide/getting-started) - First-time setup
- [Configuration](/reference/configuration) - Advanced settings
- [CLI Reference](/reference/cli) - Command documentation
