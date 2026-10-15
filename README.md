<div align="center">

# Memvault

**Local-first conversation archive and memory layer for AI agents**

Your conversations belong to you. Store them locally, search across platforms, and give agents real long-term memory.

[Installation](#installation) • [Quick Start](#quick-start) • [Documentation](docs/) • [Architecture](docs/ARCHITECTURE.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](go.mod)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

---

## What is Memvault?

Memvault is a **self-hosted conversation memory system** that:

- **Archives every conversation** from Codex, Claude Code, and OpenCode to your local disk
- **Indexes and searches** across all platforms with full-text search in English and Chinese
- **Provides agents with memory** via MCP (Model Context Protocol) tools for context-aware interactions
- **Gives you ownership** — your data stays on machines you control, readable as Markdown

Unlike cloud-based memory services like [Mem0](https://github.com/mem0ai/mem0) or [Zep](https://github.com/getzep/zep), Memvault **runs entirely on your infrastructure**. No API keys, no usage limits, no vendor lock-in.

### Why Memvault?

| Problem | Memvault's Solution |
|---------|---------------------|
| Platform conversations disappear when accounts close | Archive to local directories with checksummed objects |
| Agents forget past interactions | MCP memory tools with evidence-backed facts |
| Memory services require API keys and monthly fees | 100% local, open-source Go + React app |
| Vendor lock-in makes data migration painful | Export to ZIP with Markdown + original snapshots |
| Search across Codex + Claude Code is impossible | Unified full-text search with source filtering |

---

## Key Features

### 🗄️ **Conversation Archiving**
- Real-time capture from Codex, Claude Code, OpenCode via hooks
- Content-addressed storage (SHA-256) with automatic deduplication
- Original platform snapshots preserved alongside processed content
- Offline-safe: queued writes sync when vault directory returns

### 🔍 **Powerful Search**
- Full-text search in English and Chinese (jieba tokenization)
- Filter by source platform, role (user/assistant), memory type
- Sort by recency or relevance with complete pagination (no result truncation)
- Million-message performance benchmark included

### 🧠 **Agent Memory Layer**
- Six MCP tools: `context`, `search`, `session_read`, `memory_update`, `tasks`, `session_link`
- Evidence-backed memories with source tracing to exact messages
- Concurrent-safe memory updates with parent version tracking
- Automatic maintenance by agents (no "remember this" prompts needed)

### 📝 **Human-Readable Storage**
- Memories stored as Markdown with version comments
- JSONL journal for sessions, messages, evidence, deletions
- SQLite index is **rebuildable** — source of truth is append-only journal
- Export includes Markdown + attachments + SHA-256 manifest

### 🔐 **Privacy & Control**
- No telemetry, no cloud API, no external dependencies
- Credentials stay in machine-local config (not in shared vault)
- Vault directory can live on iCloud/Dropbox/Syncthing for multi-device sync
- Forget function shows impact scope before removing data

---

## Installation

### Homebrew (macOS / Linux)

```sh
brew install liyown/tap/memvault
```

### Shell installer (macOS / Linux / Git Bash on Windows)

```sh
curl -fsSL https://github.com/liyown/memvault/releases/latest/download/install.sh | bash
```

The installer detects your platform, downloads the latest stable release, verifies
SHA-256, and installs to `~/.local/bin`. It preserves the previous binary as
`memvault.previous` when upgrading. Add `~/.local/bin` to your PATH, then run
`memvault setup`.

To choose a version or directory, download the script and set its options:

```sh
curl -fsSL https://github.com/liyown/memvault/releases/latest/download/install.sh -o install.sh
VERSION=0.1.0 INSTALL_DIR="$HOME/.local/bin" bash install.sh
```

### Manual download

[Releases](https://github.com/liyown/memvault/releases) includes macOS and Linux
archives for ARM64 and AMD64, a Windows AMD64 ZIP, and `SHA256SUMS`. Verify the
archive against `SHA256SUMS` before extracting and installing the binary.

### Build from Source

Requires Go 1.26+, Node.js 20.19+/22.12+, npm:

```sh
git clone https://github.com/liyown/memvault.git
cd memvault
make build
make install
```

---

## Quick Start

### 1. Initialize Vault

```sh
memvault setup --root "/path/to/your/memory-vault"
```

This creates vault structure and installs hooks for Codex/Claude Code. On macOS, you'll be prompted to grant system permissions via System Settings → Privacy & Security.

### 2. Open Web UI

```sh
memvault open
```

Browse sessions, search conversations, manage memories, and configure sources.

### 3. Verify Integrations

Check the **Sources** page in the web UI to confirm:
- Hook files are installed (config written)
- Hooks have been invoked (archiving active)
- CLI clients are available (full integration)

> **Note**: Codex requires trusting the hook on first run via its native prompt. Claude clients won't auto-install if not already present.

### 4. Start Chatting

Conversations are now archived automatically. No additional commands needed.

---

## How It Works

### Storage Architecture

```
your-vault-directory/
├── memvault.json          # Vault metadata
├── objects/               # Content-addressed blobs (SHA-256)
├── journal/               # Append-only JSONL event log
├── memories/              # Human-editable Markdown
├── sessions/              # Per-session Markdown summaries
├── AGENTS.md              # Agent maintenance protocol
└── FORMAT.md              # Storage format specification
```

- **Objects**: Immutable, checksummed content (platform snapshots, messages, attachments)
- **Journal**: JSONL records of sessions, messages, memories, evidence, deletions
- **Memories**: Markdown files with version tracking for human editing
- **Index**: SQLite3 database rebuilt from journal (cache, not source of truth)

### Agent Memory Protocol

Agents access memory via [MCP tools](https://modelcontextprotocol.io/introduction):

1. **`memvault_context`** — Fetch base memories + session summaries + unprocessed tail
2. **`memvault_search`** — Full-text search with source/role filtering
3. **`memvault_session_read`** — Paginated session history with message versions
4. **`memvault_memory_update`** — Upsert facts with evidence and parent version tracking
5. **`memvault_tasks`** — Query/claim maintenance tasks (merge conflicts, staleness checks)
6. **`memvault_session_link`** — Declare continuation/branch/reference relationships

Agents maintain memories **automatically during normal conversation** — no separate "remember" command required. See [Memory Maintenance Protocol](docs/AGENT-PROTOCOL.md) for details.

### Multi-Machine Sync

Memvault does not include built-in sync. Use your preferred tool:

- **iCloud Drive / Dropbox**: Point vault to synced folder
- **Syncthing**: Peer-to-peer sync between machines
- **Git LFS**: Version-controlled vault (objects/ uses LFS)

Each machine maintains its own SQLite index. Journal entries arriving from sync are automatically indexed.

---

## CLI Reference

```sh
# Lifecycle
memvault setup --root /vault/path          # Initialize or reconfigure
memvault status                             # Show vault status and service health
memvault open                               # Launch web UI

# Search & Export
memvault search "query text"                # Search all conversations
memvault show <session-id> --format markdown  # View session as Markdown
memvault export --out archive.zip           # Export vault to portable ZIP
memvault import archive.zip                 # Import from exported archive

# Maintenance
memvault repair --action verify             # Check journal + object integrity
memvault repair --action rebuild            # Rebuild SQLite index from journal
memvault repair --action integrations       # Re-install hooks for Codex/Claude

# Advanced
memvault serve                              # Run server in foreground (no daemon)
memvault mcp                                # MCP stdio mode for direct agent integration
memvault uninstall                          # Remove integrations (keeps vault)
```

---

## Comparison to Alternatives

| Feature | Memvault | [Mem0](https://github.com/mem0ai/mem0) | [Zep](https://github.com/getzep/zep) | [LangChain Memory](https://github.com/langchain-ai/langmem) |
|---------|----------|---------|-----|--------------|
| **Deployment** | Self-hosted | Cloud or self-hosted | Cloud or self-hosted | Self-hosted |
| **Storage** | Local files (Markdown + JSONL) | Vector DB + SQL | PostgreSQL + graph | In-memory or custom |
| **Platform Coverage** | Codex, Claude Code, OpenCode | API-based (any) | API-based (any) | LangChain apps only |
| **Archive Format** | Human-readable + checksummed objects | Proprietary | SQL dumps | Code-dependent |
| **Multi-Device** | User-managed sync | Cloud-native | Cloud-native | Not designed for it |
| **Cost** | Free (MIT) | Free tier + paid plans | Free tier + paid plans | Free (MIT) |
| **Primary Use Case** | Platform-agnostic conversation archive | Personalized AI assistants | Production chatbots with memory | LangChain agent memory |

**Choose Memvault if you want**:
- Permanent, readable archives of conversations from multiple platforms
- Full data ownership without cloud dependencies
- Agent memory that survives platform shutdowns

**Choose cloud services if you want**:
- Managed infrastructure with no setup
- Multi-user SaaS with team collaboration
- Vendor support and uptime SLAs

---

## Performance

Tested on M1 MacBook Pro (2020, 16GB):

| Benchmark | Result | Details |
|-----------|--------|---------|
| **Index 1M messages** | ~45s | 10,000 sessions, cold start |
| **Full-text search** | <100ms | Typical query on 1M corpus |
| **Relevance-ranked search** | <2s | Sorted by score (1M corpus) |
| **Memory context retrieval** | <50ms | Base memories + summaries + tail |

See [`internal/app/search_perf_test.go`](internal/app/search_perf_test.go) for reproducible benchmarks.

---

## Documentation

- **[Architecture Overview](docs/ARCHITECTURE.md)** — System design, modules, data flow
- **[Agent Protocol](docs/AGENT-PROTOCOL.md)** — Memory maintenance guidelines for agents
- **[Storage Format](docs/FORMAT.md)** — JSONL schema, object addressing, versioning
- **[Validation Report](docs/VALIDATION.md)** — Test coverage, known limits, roadmap
- **[UI Implementation](docs/UI-DETAILS.md)** — Frontend architecture and design system
- **[Release Process](docs/RELEASE.md)** — CI/CD, versioning, Homebrew packaging

---

## Roadmap

**0.1.x** (Current)
- ✅ Codex/Claude Code/OpenCode archiving
- ✅ Full-text search (EN/ZH) with source filtering
- ✅ MCP memory tools for agents
- ✅ Markdown export + ZIP archives
- ✅ Native folder picker (macOS)

**0.2.x** (Planned)
- [ ] Attachment handling (images, PDFs in conversations)
- [ ] Incremental journal compaction
- [ ] Multi-vault support (work, personal, research)
- [ ] Graph view of session relationships

**1.0** (Target)
- [ ] Windows support (folder picker, service, hooks)
- [ ] Mobile-responsive web UI
- [ ] Plugin system for custom sources
- [ ] 100% test coverage on core paths

See [VALIDATION.md](docs/VALIDATION.md) for detailed acceptance criteria.

---

## Contributing

Contributions welcome! Please:

1. Read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines
2. Check [open issues](https://github.com/liyown/memvault/issues) for tasks
3. Run `make test` before submitting PRs
4. Follow Go + TypeScript coding standards in [`.claude/rules/`](.claude/rules/)

---

## License

[MIT](LICENSE) — you own your code, you own your data.

---

## Acknowledgments

- **[Mem0](https://github.com/mem0ai/mem0)** and **[Zep](https://github.com/getzep/zep)** for pioneering agent memory systems
- **[LangChain Memory](https://github.com/langchain-ai/langmem)** for conversation history patterns
- **[Model Context Protocol](https://modelcontextprotocol.io/)** for standardized tool interfaces
- **Codex**, **Claude Code**, **OpenCode** teams for building extensible agent platforms

---

<div align="center">

**Built with Go + React** | **100% Local** | **Open Source**

[⬆ Back to top](#memvault)

</div>
