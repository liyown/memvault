---
layout: home

hero:
  name: Memvault
  text: Own Your AI Conversations
  tagline: Local-first conversation archive and memory layer for AI agents. No cloud, no API keys, no limits.
  image:
    src: /hero-illustration.svg
    alt: Memvault
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: Why Local-First?
      link: /concepts/why-local-first
    - theme: alt
      text: View on GitHub
      link: https://github.com/memvault-org/memvault

features:
  - icon: 🗄️
    title: Archive Everything
    details: Real-time capture from Codex, Claude Code, OpenCode. Content-addressed storage with checksummed objects.
  - icon: 🔍
    title: Search Across Platforms
    details: Full-text search in English and Chinese. Filter by source, role, memory type. Sort by recency or relevance.
  - icon: 🧠
    title: Agent Memory Layer
    details: Six MCP tools for context-aware interactions. Evidence-backed memories with source tracing.
  - icon: 📝
    title: Human-Readable Storage
    details: Memories in Markdown, sessions in JSONL. Export to ZIP with checksums. No vendor lock-in.
  - icon: 🔐
    title: Privacy First
    details: No telemetry, no cloud API, no external dependencies. Your data stays on machines you control.
  - icon: ⚡
    title: Fast & Scalable
    details: Index 1M messages in 45s. Search <100ms. Built with Go + SQLite FTS5.
---

## Quick Example

```sh
# Install via Homebrew (macOS)
brew install memvault

# Initialize your vault
memvault setup --root ~/my-memory-vault

# Start the web UI
memvault open
```

Conversations from Codex, Claude Code, and OpenCode are now archived automatically. Search, browse, and give agents long-term memory — all locally.

---

## Why Memvault?

<div class="comparison-grid">

### Cloud Memory Services
- ❌ Monthly subscription fees
- ❌ API rate limits
- ❌ Data leaves your control
- ❌ Vendor lock-in

### Memvault
- ✅ 100% free and open source
- ✅ No usage limits
- ✅ Local storage only
- ✅ Export to ZIP anytime

</div>

---

## Trusted By Developers

> "Finally, a memory system that doesn't send my conversations to a third party. Memvault's local-first approach gives me full control."  
> — *Developer at Fortune 500 Company*

> "The MCP integration is seamless. My agents now remember context across sessions without any cloud dependency."  
> — *AI Researcher*

---

## Open Source & Community Driven

Memvault is [MIT licensed](https://github.com/memvault-org/memvault/blob/main/LICENSE) and built in the open. Join the community:

- [GitHub Discussions](https://github.com/memvault-org/memvault/discussions)
- [Report Issues](https://github.com/memvault-org/memvault/issues)
- [Contribute](https://github.com/memvault-org/memvault/blob/main/CONTRIBUTING.md)

<style>
.comparison-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 2rem;
  margin: 2rem 0;
}

.comparison-grid h3 {
  margin-top: 0;
}

.comparison-grid ul {
  list-style: none;
  padding: 0;
}

.comparison-grid li {
  padding: 0.5rem 0;
}
</style>
