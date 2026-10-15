#!/usr/bin/env bash
# Install a checksum-verified release, or an explicitly versioned local package.
set -euo pipefail

REPO="${REPO:-liyown/memvault}"
VERSION="${VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
DIST_DIR="${DIST_DIR:-dist}"
BASE_URL="${BASE_URL:-https://github.com/${REPO}/releases/download}"
API_URL="${API_URL:-https://api.github.com/repos/${REPO}/releases/latest}"
TEMP_DIR=""
trap 'if [ -n "$TEMP_DIR" ]; then rm -rf "$TEMP_DIR"; fi' EXIT

fail() { echo "Error: $*" >&2; exit 1; }

detect_platform() {
  local os arch
  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux) os="linux" ;;
    MINGW*|MSYS*|CYGWIN*) os="windows" ;;
    *) fail "Unsupported OS: $(uname -s)" ;;
  esac
  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) fail "Unsupported architecture: $(uname -m)" ;;
  esac
  [ "$os/$arch" != "windows/arm64" ] || fail "Windows ARM64 packages are not available"
  echo "${os}_${arch}"
}

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    fail "sha256sum or shasum is required"
  fi
}

main() {
  local platform archive binary expected actual
  platform="$(detect_platform)"
  TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/memvault-install.XXXXXX")"
  if [ -z "$VERSION" ]; then
    echo "==> Finding latest release"
    curl -fsSL --retry 3 "$API_URL" -o "$TEMP_DIR/release.json"
    VERSION="$(sed -nE 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/p' "$TEMP_DIR/release.json" | head -1)"
    [ -n "$VERSION" ] || fail "Could not determine the latest release; set VERSION explicitly"
  fi
  VERSION="${VERSION#v}"
  [[ "$VERSION" =~ ^[0-9A-Za-z][0-9A-Za-z.+-]*$ ]] || fail "Invalid version: $VERSION"
  binary="memvault"
  archive="memvault_${VERSION}_${platform}.tar.gz"
  if [ "$platform" = "windows_amd64" ]; then
    binary="memvault.exe"
    archive="memvault_${VERSION}_${platform}.zip"
  fi
  echo "==> Installing memvault ${VERSION} for ${platform}"
  if [ -f "$DIST_DIR/$archive" ]; then
    [ -f "$DIST_DIR/SHA256SUMS" ] || fail "Local package is missing SHA256SUMS"
    cp "$DIST_DIR/$archive" "$TEMP_DIR/$archive"
    cp "$DIST_DIR/SHA256SUMS" "$TEMP_DIR/SHA256SUMS"
  else
    curl -fsSL --retry 3 "${BASE_URL}/v${VERSION}/${archive}" -o "$TEMP_DIR/$archive"
    curl -fsSL --retry 3 "${BASE_URL}/v${VERSION}/SHA256SUMS" -o "$TEMP_DIR/SHA256SUMS"
  fi
  expected="$(awk -v name="$archive" '$2 == name || $2 == "*" name {print $1}' "$TEMP_DIR/SHA256SUMS")"
  [[ "$expected" =~ ^[0-9a-fA-F]{64}$ ]] || fail "Missing or invalid checksum for $archive"
  actual="$(checksum "$TEMP_DIR/$archive")"
  [ "$actual" = "$expected" ] || fail "SHA-256 mismatch for $archive; existing installation was preserved"
  if [ "$platform" = "windows_amd64" ]; then
    unzip -q "$TEMP_DIR/$archive" "$binary" -d "$TEMP_DIR"
  else
    tar xzf "$TEMP_DIR/$archive" -C "$TEMP_DIR" "$binary"
  fi
  [ -f "$TEMP_DIR/$binary" ] || fail "Package does not contain $binary"
  mkdir -p "$INSTALL_DIR"
  if [ -f "$INSTALL_DIR/$binary" ]; then
    cp "$INSTALL_DIR/$binary" "$INSTALL_DIR/$binary.previous"
  fi
  cp "$TEMP_DIR/$binary" "$INSTALL_DIR/$binary.installing"
  chmod 755 "$INSTALL_DIR/$binary.installing"
  mv -f "$INSTALL_DIR/$binary.installing" "$INSTALL_DIR/$binary"
  echo "==> Installed $INSTALL_DIR/$binary"
  echo "Run 'memvault setup' to initialize. Ensure $INSTALL_DIR is on your PATH."
}

main "$@"
