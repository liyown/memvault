# Release Process

The public repository is [liyown/memvault](https://github.com/liyown/memvault).
Homebrew packages live in [liyown/homebrew-tap](https://github.com/liyown/homebrew-tap).

## Publish a version

Wait for CI on the intended main commit to pass, then create a semantic version tag:

```sh
git tag -a v0.1.0 -m 'Release v0.1.0'
git push origin v0.1.0
```

The Release workflow tests the code and installer, builds all five platform packages,
checks a local Linux installation, and publishes these assets:

- macOS ARM64 and AMD64 `.tar.gz` archives
- Linux ARM64 and AMD64 `.tar.gz` archives
- Windows AMD64 `.zip` archive
- `SHA256SUMS`, `install.sh`, and a generated `memvault.rb` formula

For stable tags, a second job downloads the published formula and commits it to
`Formula/memvault.rb` in the tap. The `HOMEBREW_TAP_DEPLOY_KEY` repository secret
holds an SSH deploy key with write access to that tap only. Its public key is
registered in the tap repository's deploy keys. No personal access token is needed.
If the tap update fails after publication, fix the problem and rerun the failed job.

Tags with a suffix, such as `v0.2.0-rc.1`, create prereleases and do not update the
stable Homebrew formula. Install prereleases by explicitly setting `VERSION`.

## Install and verify

```sh
brew install liyown/tap/memvault
brew test liyown/tap/memvault

curl -fsSL https://github.com/liyown/memvault/releases/latest/download/install.sh -o install.sh
INSTALL_DIR="$(mktemp -d)/bin" bash install.sh
```

The installer checks SHA-256 before replacing a binary. An explicit `VERSION=0.1.0`
selects that release. If `DIST_DIR` (default `dist`) contains the matching archive
and `SHA256SUMS`, it installs locally without downloading the package. Otherwise,
it downloads from GitHub. `REPO`, `BASE_URL`, and `API_URL` can override the source
for forks and local testing.

## Build packages locally

```sh
VERSION=0.1.0 make dist
./scripts/brew-formula.sh 0.1.0
VERSION=0.1.0 INSTALL_DIR="$(mktemp -d)/bin" bash scripts/install.sh
```

Only archives for the current version are included in `SHA256SUMS`; old local
packages are not published by the clean CI checkout. `scripts/brew-formula.sh`
uses those same archives to produce `dist/memvault.rb`. The source template is
`homebrew/memvault.rb.in`; it is not an installable formula.

Versions are injected at build time with
`-X memvault/internal/app.Version=<version>`. Verify with `memvault version`.
