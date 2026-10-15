# Memvault Documentation

This directory contains the source for the Memvault documentation site, built with [VitePress](https://vitepress.dev/).

## Development

```sh
cd docs-site

# Install dependencies
pnpm install

# Start dev server
pnpm dev

# Build for production
pnpm build

# Preview production build
pnpm preview
```

## Structure

```
docs-site/
├── .vitepress/
│   ├── config.ts           # VitePress configuration
│   └── theme/
│       ├── custom.css      # Custom styles
│       └── index.ts        # Theme entry
├── concepts/               # Conceptual guides
│   ├── conversation-vs-memory.md
│   ├── why-local-first.md
│   ├── data-sovereignty.md
│   ├── mcp.md
│   └── storage.md
├── guide/                  # Step-by-step tutorials
│   ├── introduction.md
│   ├── installation.md
│   ├── getting-started.md
│   └── ...
├── reference/              # API & CLI reference
│   ├── cli.md
│   ├── mcp-tools.md
│   └── configuration.md
├── public/                 # Static assets
│   ├── hero-illustration.svg
│   └── logo.svg
└── index.md                # Homepage
```

## Deployment

Documentation is automatically deployed to GitHub Pages on every push to `main` via `.github/workflows/deploy-docs.yml`.

**Live URL**: https://memvault-org.github.io/memvault/

## Contributing

See the main [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

### Writing Documentation

- Use clear, concise language
- Include code examples
- Add Mermaid diagrams for complex concepts
- Link to related pages
- Follow E-E-A-T principles (add citations)

### Style Guide

- **Headings**: Sentence case ("Why local-first?", not "Why Local-First?")
- **Code blocks**: Include language identifier
- **Links**: Relative for internal, absolute for external
- **Images**: SVG preferred (hero illustrations, diagrams)

## License

MIT
