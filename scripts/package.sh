#!/usr/bin/env bash
# Build and checksum the five supported release archives.
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-preview")}"
VERSION="${VERSION#v}"
[[ "$VERSION" =~ ^[0-9A-Za-z][0-9A-Za-z.+-]*$ ]] || { echo "Invalid version" >&2; exit 1; }
DIST_DIR="${DIST_DIR:-dist}"
mkdir -p "$DIST_DIR"
DIST_DIR="$(cd "$DIST_DIR" && pwd)"
BUILD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/memvault-build.XXXXXX")"
trap 'rm -rf "$BUILD_DIR"' EXIT
platforms=(darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64)
archives=()
echo "==> Building frontend"
(cd web && npm ci --no-audit --no-fund && npm run build)
for platform in "${platforms[@]}"; do
  IFS=/ read -r os arch <<< "$platform"
  binary=memvault
  [ "$os" != windows ] || binary=memvault.exe
  echo "==> Building ${os}/${arch} (${VERSION})"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X memvault/internal/app.Version=${VERSION}" \
    -o "$BUILD_DIR/$binary" ./cmd/memvault
  archive="memvault_${VERSION}_${os}_${arch}"
  if [ "$os" = windows ]; then
    archive="${archive}.zip"
    # Remove a previous archive so zip never retains stale members.
    rm -f "$DIST_DIR/$archive"
    (cd "$BUILD_DIR" && zip -q "$DIST_DIR/$archive" "$binary")
  else
    archive="${archive}.tar.gz"
    tar czf "$DIST_DIR/$archive" -C "$BUILD_DIR" "$binary"
  fi
  archives+=("$archive")
  rm "$BUILD_DIR/$binary"
done
(cd "$DIST_DIR" && if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${archives[@]}"
else
  shasum -a 256 "${archives[@]}"
fi) > "$DIST_DIR/SHA256SUMS"
echo "==> Packages and SHA256SUMS written to $DIST_DIR"
