# Release Process

## Automated Release via GitHub Actions

### Prerequisites
1. Ensure all tests pass: `go test ./...`
2. Build and verify locally: `make dist`
3. Update version if needed in code

### Create Release

```bash
# Tag the release
git tag -a v0.1.0 -m "Release v0.1.0"

# Push the tag to trigger CI
git push origin v0.1.0
```

The GitHub Actions workflow will:
- Build all platform binaries
- Generate checksums
- Create a GitHub release with artifacts
- Attach Darwin (macOS), Linux, and Windows builds

### Update Homebrew Formula

After the GitHub release is created:

```bash
# Generate the formula (using the built artifacts)
./scripts/brew-formula.sh 0.1.0

# The script outputs: homebrew/memvault-0.1.0.rb
```

Then update the tap repository:

```bash
# Clone or navigate to homebrew-tap repo
git clone git@github.com:memvault-org/homebrew-tap.git
cd homebrew-tap

# Copy the generated formula
cp ../memvault/homebrew/memvault-0.1.0.rb Formula/memvault.rb

# Commit and push
git add Formula/memvault.rb
git commit -m "memvault: update to 0.1.0"
git push origin main
```

Users can then install via:
```bash
brew tap memvault-org/tap
brew install memvault
```

## Manual Release

If you need to release without CI:

```bash
# Set version
export VERSION=0.1.0

# Build all platforms
make dist

# Generate Homebrew formula
./scripts/brew-formula.sh $VERSION

# Create GitHub release manually and upload dist/*.tar.gz and dist/*.zip
```

## Version Injection

The build system injects the version at compile time:

```bash
go build -ldflags "-X memvault/internal/app.Version=0.1.0" ./cmd/memvault
```

Verify with:
```bash
./memvault version
```
