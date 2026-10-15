#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 0.1.0"
  exit 1
fi

DIST_DIR="dist"
FORMULA_TEMPLATE="homebrew/memvault.rb"
FORMULA_OUT="homebrew/memvault-${VERSION}.rb"

echo "==> Generating Homebrew formula for version ${VERSION}"

# Calculate SHA256 for each platform
echo "==> Calculating checksums"
SHA_DARWIN_ARM64=$(sha256sum "${DIST_DIR}/memvault_${VERSION}_darwin_arm64.tar.gz" | awk '{print $1}')
SHA_DARWIN_AMD64=$(sha256sum "${DIST_DIR}/memvault_${VERSION}_darwin_amd64.tar.gz" | awk '{print $1}')
SHA_LINUX_ARM64=$(sha256sum "${DIST_DIR}/memvault_${VERSION}_linux_arm64.tar.gz" | awk '{print $1}')
SHA_LINUX_AMD64=$(sha256sum "${DIST_DIR}/memvault_${VERSION}_linux_amd64.tar.gz" | awk '{print $1}')

echo "Darwin ARM64: ${SHA_DARWIN_ARM64}"
echo "Darwin AMD64: ${SHA_DARWIN_AMD64}"
echo "Linux ARM64:  ${SHA_LINUX_ARM64}"
echo "Linux AMD64:  ${SHA_LINUX_AMD64}"

# Generate formula
sed \
  -e "s/VERSION_PLACEHOLDER/${VERSION}/g" \
  -e "s/SHA256_ARM64_PLACEHOLDER/${SHA_DARWIN_ARM64}/" \
  -e "s/SHA256_AMD64_PLACEHOLDER/${SHA_DARWIN_AMD64}/" \
  -e "s/SHA256_LINUX_ARM64_PLACEHOLDER/${SHA_LINUX_ARM64}/" \
  -e "s/SHA256_LINUX_AMD64_PLACEHOLDER/${SHA_LINUX_AMD64}/" \
  "${FORMULA_TEMPLATE}" > "${FORMULA_OUT}"

echo "==> Formula written to ${FORMULA_OUT}"
echo ""
echo "To publish:"
echo "  1. Create/update tap repository: memvault-org/homebrew-tap"
echo "  2. Copy ${FORMULA_OUT} to Formula/memvault.rb in tap repo"
echo "  3. Commit and push"
echo ""
echo "Users can then install with:"
echo "  brew tap memvault-org/tap"
echo "  brew install memvault"
