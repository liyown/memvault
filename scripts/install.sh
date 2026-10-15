#!/usr/bin/env bash
# Install memvault from local dist or download from release
set -euo pipefail

VERSION="${VERSION:-0.1.0-preview}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BASE_URL="${BASE_URL:-https://github.com/liuyaowen/memvault/releases/download}"

detect_platform() {
  local os arch
  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    MINGW*|MSYS*|CYGWIN*) os="windows" ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac

  echo "${os}_${arch}"
}

main() {
  local platform
  platform="$(detect_platform)"
  local archive_name="memvault_${VERSION}_${platform}"

  echo "==> Installing memvault ${VERSION} for ${platform}"

  # Try local dist first
  if [ -f "dist/${archive_name}.tar.gz" ]; then
    echo "==> Found local package"
    tar xzf "dist/${archive_name}.tar.gz" -C /tmp
  elif [ -f "dist/${archive_name}.zip" ]; then
    echo "==> Found local package"
    unzip -q "dist/${archive_name}.zip" -d /tmp
  else
    # Download from release
    local url
    if [ "$platform" = "windows_amd64" ]; then
      url="${BASE_URL}/v${VERSION}/${archive_name}.zip"
      echo "==> Downloading ${url}"
      curl -fsSL -o "/tmp/${archive_name}.zip" "$url"
      unzip -q "/tmp/${archive_name}.zip" -d /tmp
      rm "/tmp/${archive_name}.zip"
    else
      url="${BASE_URL}/v${VERSION}/${archive_name}.tar.gz"
      echo "==> Downloading ${url}"
      curl -fsSL "$url" | tar xz -C /tmp
    fi
  fi

  # Install
  mkdir -p "$INSTALL_DIR"
  local binary="/tmp/memvault"
  [ "$platform" = "windows_amd64" ] && binary="/tmp/memvault.exe"

  if [ -f "${INSTALL_DIR}/memvault" ]; then
    echo "==> Backing up existing binary"
    cp "${INSTALL_DIR}/memvault" "${INSTALL_DIR}/memvault.previous"
  fi

  echo "==> Installing to ${INSTALL_DIR}"
  cp "$binary" "${INSTALL_DIR}/memvault.installing"
  chmod 755 "${INSTALL_DIR}/memvault.installing"
  mv "${INSTALL_DIR}/memvault.installing" "${INSTALL_DIR}/memvault"
  rm -f "$binary"

  echo "==> memvault installed successfully"
  echo ""
  echo "Run 'memvault setup' to initialize"
  echo ""
  echo "Add ${INSTALL_DIR} to PATH if not already:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
}

main "$@"
