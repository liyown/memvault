# Contributing to Memvault

Thank you for considering contributing to Memvault! This document provides guidelines for contributing code, documentation, bug reports, and feature requests.

## Code of Conduct

Be respectful and constructive. Personal attacks, harassment, and discriminatory behavior will not be tolerated.

## How to Contribute

### Reporting Bugs

Before opening an issue:

1. **Search existing issues** to avoid duplicates
2. **Check the documentation** — the issue might be covered in [docs/](docs/)
3. **Test on the latest version** from `main` branch

When filing a bug report, include:

- **Version**: Output of `memvault version`
- **Platform**: OS (macOS/Linux/Windows), architecture (amd64/arm64)
- **Steps to reproduce**: Minimal commands to trigger the issue
- **Expected vs actual behavior**
- **Logs**: Relevant output from `memvault status` or server logs

### Suggesting Features

Feature requests should include:

- **Use case**: What problem does this solve?
- **Proposed solution**: How should it work?
- **Alternatives considered**: What other approaches did you think about?
- **Impact**: Who benefits? How widely needed is this?

Open an issue labeled `enhancement` to discuss before implementing.

## Development Setup

### Prerequisites

- **Go**: 1.23 or later
- **Node.js**: 20.19+ or 22.12+
- **npm**: Latest stable
- **Make**: GNU Make or compatible

### Clone and Build

```sh
git clone https://github.com/memvault-org/memvault.git
cd memvault

# Install dependencies
go mod download
cd web && npm ci && cd ..

# Build
make build

# Run tests
make test

# Install locally
make install
```

### Project Structure

```
memvault/
├── cmd/memvault/           # CLI entry point
├── internal/
│   ├── app/                # Core application logic
│   ├── archive/            # Conversation archiving
│   ├── memory/             # Memory storage and retrieval
│   ├── search/             # Full-text search (jieba, SQLite FTS5)
│   ├── mcp/                # MCP server implementation
│   └── integration/        # Platform hooks (Codex, Claude Code)
├── web/
│   ├── src/                # React + TypeScript frontend
│   └── dist/               # Build output (embedded in Go binary)
├── scripts/                # Build, install, CI scripts
├── docs/                   # Architecture and design docs
└── .claude/rules/          # Code style guidelines
```

### Code Style

- **Go**: Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
  - Run `go fmt`, `go vet`, `staticcheck` before committing
  - See [`.claude/rules/golang/`](.claude/rules/golang/) for project conventions
- **TypeScript**: Follow [`.claude/rules/typescript/`](.claude/rules/typescript/)
  - Use Prettier for formatting
  - No `console.log` in production code
- **Tests**: Table-driven tests preferred (Go), Jest + React Testing Library (TypeScript)

### Running Tests

```sh
# All tests
make test

# Go tests only
go test -v -race ./...

# Frontend tests only
cd web && npm test

# Performance benchmarks
MEMVAULT_PERF=1 go test ./internal/app -run TestMillionMessageSearch -v -timeout 20m
```

## Pull Request Process

### Before Submitting

1. **Branch from `main`**: `git checkout -b feature/my-feature`
2. **Write tests**: All new code should have tests
3. **Update docs**: Add/update relevant files in `docs/`
4. **Run full test suite**: `make test`
5. **Check for linter errors**: `make lint` (if configured)
6. **Commit with clear messages**: See commit message format below

### Commit Message Format

```
type(scope): subject

body (optional)

footer (optional)
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`

**Example**:
```
feat(search): add relevance ranking for full-text search

Implements BM25 scoring for result ordering. Users can now sort
by relevance in addition to recency.

Closes #42
```

### Pull Request Template

Title should follow commit message format. In the description:

- **What**: Describe the changes
- **Why**: Motivation and context
- **How**: Approach taken
- **Testing**: How did you verify the changes?
- **Screenshots**: If UI changes, include before/after
- **Checklist**:
  - [ ] Tests added/updated
  - [ ] Documentation updated
  - [ ] `make test` passes
  - [ ] Commit messages follow guidelines

### Review Process

1. CI checks must pass (tests, build)
2. At least one maintainer approval required
3. Address reviewer feedback with additional commits (no force-push until approved)
4. Maintainer will squash/merge once approved

## Areas for Contribution

### Good First Issues

Check issues labeled [`good first issue`](https://github.com/memvault-org/memvault/labels/good%20first%20issue) for beginner-friendly tasks:

- Documentation improvements
- Error message clarity
- Test coverage gaps
- CLI help text updates

### High-Impact Areas

- **Attachment handling**: Images, PDFs in conversations (see [VALIDATION.md](docs/VALIDATION.md))
- **Windows support**: Native folder picker, service, hooks
- **Mobile UI**: Responsive web design for phone screens
- **Performance**: Optimize search for 10M+ message corpora
- **Platform integrations**: New sources beyond Codex/Claude/OpenCode

### Documentation Needs

- Translation to other languages
- Tutorial videos or blog posts
- Example workflows and use cases
- Troubleshooting guides

## Testing Guidelines

### Test Coverage Expectations

- **Core modules** (`internal/app`, `internal/memory`): >80% coverage
- **Integration tests** (`internal/integration`): Key happy paths + error cases
- **Frontend**: Component tests for interactive UI, snapshots for layout

### Writing Good Tests

```go
// Table-driven test example
func TestMemoryUpdate(t *testing.T) {
    tests := []struct {
        name    string
        input   Memory
        want    error
        wantErr bool
    }{
        {
            name:    "valid memory",
            input:   Memory{Title: "Test", Kind: "knowledge"},
            want:    nil,
            wantErr: false,
        },
        // ... more cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := UpdateMemory(tt.input)
            if (got != nil) != tt.wantErr {
                t.Errorf("UpdateMemory() error = %v, wantErr %v", got, tt.wantErr)
            }
        })
    }
}
```

## Release Process

Maintainers handle releases. For contributors:

- Releases follow [Semantic Versioning](https://semver.org/)
- Version tags trigger CI to build and publish artifacts
- Changelog is auto-generated from commit messages
- Homebrew formula updated after GitHub release

See [docs/RELEASE.md](docs/RELEASE.md) for full release workflow.

## Questions?

- **GitHub Discussions**: For general questions and ideas
- **Issues**: For bug reports and feature requests
- **Email**: For security vulnerabilities (do not open public issues)

---

**Thank you for contributing to Memvault!** 🙏
