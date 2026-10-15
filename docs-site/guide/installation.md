# Installation

Memvault is distributed as a single binary for macOS and Linux. Windows support is coming soon.

## System Requirements

- macOS 11+ or Linux
- 2GB free disk space
- One of: Codex, Claude Code, or OpenCode

## macOS

### Homebrew (Recommended)

```sh
brew tap memvault-org/tap
brew install memvault
```

Verify installation:

```sh
memvault version
# memvault 0.1.0
```

### Manual Install

1. Download the latest release:

```sh
curl -LO https://github.com/memvault-org/memvault/releases/latest/download/memvault_0.1.0_darwin_arm64.tar.gz
```

2. Extract and install:

```sh
tar xzf memvault_0.1.0_darwin_arm64.tar.gz
sudo mv memvault /usr/local/bin/
```

3. Grant permissions:

System Preferences → Security & Privacy → Allow `memvault` to run

---

## Linux

### Arch Linux (AUR)

```sh
yay -S memvault-bin
```

### Debian/Ubuntu

```sh
# Download .deb package
curl -LO https://github.com/memvault-org/memvault/releases/latest/download/memvault_0.1.0_amd64.deb

# Install
sudo dpkg -i memvault_0.1.0_amd64.deb
```

### Fedora/RHEL

```sh
# Download .rpm package
curl -LO https://github.com/memvault-org/memvault/releases/latest/download/memvault_0.1.0_amd64.rpm

# Install
sudo rpm -i memvault_0.1.0_amd64.rpm
```

### Generic Binary

```sh
# Download
curl -LO https://github.com/memvault-org/memvault/releases/latest/download/memvault_0.1.0_linux_amd64.tar.gz

# Extract
tar xzf memvault_0.1.0_linux_amd64.tar.gz

# Install
sudo mv memvault /usr/local/bin/
```

---

## Build from Source

Requires Go 1.23+:

```sh
git clone https://github.com/memvault-org/memvault.git
cd memvault
make build

# Binary at ./bin/memvault
sudo mv bin/memvault /usr/local/bin/
```

---

## Post-Install Setup

Run the setup wizard:

```sh
memvault setup
```

This will:

1. Create vault directory (default: `~/memvault`)
2. Install hooks for Codex/Claude Code/OpenCode
3. Initialize SQLite index
4. Generate config file

---

## Verify Installation

```sh
# Check version
memvault version

# Check sources
memvault sources

# Open web UI
memvault open
```

Expected output:

```
Source        Hook   CLI    Last Scan
codex         ✅     ✅     2026-01-15 10:30
claude        ✅     ✅     —
opencode      ✅     ❌     —
```

---

## Permissions (macOS)

Memvault requires:

- **Accessibility**: To install hooks
- **Full Disk Access**: To read agent directories

Grant in: System Preferences → Security & Privacy → Privacy

---

## Uninstall

### macOS (Homebrew)

```sh
brew uninstall memvault
brew untap memvault-org/tap

# Remove vault (optional)
rm -rf ~/memvault
```

### Linux

```sh
# Debian/Ubuntu
sudo dpkg -r memvault

# Fedora/RHEL
sudo rpm -e memvault

# Manual
sudo rm /usr/local/bin/memvault

# Remove vault (optional)
rm -rf ~/memvault
```

---

## Next Steps

- [Getting Started](/guide/getting-started) — First-time setup
- [Configuration](/reference/configuration) — Config options
- [Troubleshooting](/guide/troubleshooting) — Common issues
