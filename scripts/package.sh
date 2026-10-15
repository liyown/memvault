#!/usr/bin/env bash
# Package memvault for distribution across platforms
set -euo pipefail

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-preview")}"
DIST_DIR="dist"
BUILD_DIR="build"

platforms=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/arm64"
  "linux/amd64"
  "windows/amd64"
)

echo "==> Building memvault ${VERSION}"
mkdir -p "${DIST_DIR}" "${BUILD_DIR}"

# Build frontend once
echo "==> Building frontend"
cd web
npm ci --no-audit --no-fund >/dev/null
npm run build
cd ..

# Build for each platform
for platform in "${platforms[@]}"; do
  IFS='/' read -r os arch <<< "$platform"

  output_name="memvault"
  if [ "$os" = "windows" ]; then
    output_name="memvault.exe"
  fi

  echo "==> Building ${os}/${arch}"
  GOOS="$os" GOARCH="$arch" go build \
    -trimpath \
    -ldflags "-s -w -X memvault/internal/app.Version=${VERSION}" \
    -o "${BUILD_DIR}/${output_name}" \
    ./cmd/memvault

  # Package
  archive_name="memvault_${VERSION}_${os}_${arch}"
  if [ "$os" = "windows" ]; then
    (cd "${BUILD_DIR}" && zip -q "../${DIST_DIR}/${archive_name}.zip" "${output_name}")
  else
    tar czf "${DIST_DIR}/${archive_name}.tar.gz" -C "${BUILD_DIR}" "${output_name}"
  fi

  rm "${BUILD_DIR}/${output_name}"
done

# Generate checksums
echo "==> Generating checksums"
cd "${DIST_DIR}"
shasum -a 256 memvault_* > SHA256SUMS
cd ..

# Cleanup
rm -rf "${BUILD_DIR}"

echo "==> Done"
ls -lh "${DIST_DIR}"
