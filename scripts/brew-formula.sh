#!/usr/bin/env bash
# Generate the tap formula from exactly the archives published in this release.
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:?Usage: scripts/brew-formula.sh <version> [output]}"
VERSION="${VERSION#v}"
[[ "$VERSION" =~ ^[0-9A-Za-z][0-9A-Za-z.+-]*$ ]] || { echo "Invalid version" >&2; exit 1; }
DIST_DIR="${DIST_DIR:-dist}"
FORMULA_OUT="${2:-${DIST_DIR}/memvault.rb}"
checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}
SHA_DARWIN_ARM64=$(checksum "${DIST_DIR}/memvault_${VERSION}_darwin_arm64.tar.gz")
SHA_DARWIN_AMD64=$(checksum "${DIST_DIR}/memvault_${VERSION}_darwin_amd64.tar.gz")
SHA_LINUX_ARM64=$(checksum "${DIST_DIR}/memvault_${VERSION}_linux_arm64.tar.gz")
SHA_LINUX_AMD64=$(checksum "${DIST_DIR}/memvault_${VERSION}_linux_amd64.tar.gz")
mkdir -p "$(dirname "$FORMULA_OUT")"
sed \
  -e "s/VERSION_PLACEHOLDER/${VERSION}/g" \
  -e "s/SHA256_ARM64_PLACEHOLDER/${SHA_DARWIN_ARM64}/" \
  -e "s/SHA256_AMD64_PLACEHOLDER/${SHA_DARWIN_AMD64}/" \
  -e "s/SHA256_LINUX_ARM64_PLACEHOLDER/${SHA_LINUX_ARM64}/" \
  -e "s/SHA256_LINUX_AMD64_PLACEHOLDER/${SHA_LINUX_AMD64}/" \
  homebrew/memvault.rb.in > "$FORMULA_OUT"
echo "Generated $FORMULA_OUT for liyown/homebrew-tap"
