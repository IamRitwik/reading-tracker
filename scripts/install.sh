#!/usr/bin/env sh
# install.sh — download and install reading-tracker.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/IamRitwik/reading-tracker/main/scripts/install.sh | sh
#   curl -fsSL ... | sh -s -- --version v0.1.0
#   curl -fsSL ... | sh -s -- --dir ~/bin
#
# Environment:
#   RT_VERSION  — override version (default: latest)
#   RT_DIR      — install directory (default: ~/.local/bin)

set -eu

REPO="IamRitwik/reading-tracker"
BINARY="reading-tracker"
VERSION="${RT_VERSION:-}"
INSTALL_DIR="${RT_DIR:-$HOME/.local/bin}"

# --- Parse args ----------------------------------------------------------
while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --dir)     INSTALL_DIR="$2"; shift 2 ;;
    --help)
      echo "Usage: install.sh [--version vX.Y.Z] [--dir /path]"
      exit 0
      ;;
    *) echo "unknown option: $1" >&2; exit 1 ;;
  esac
done

# --- Detect platform -----------------------------------------------------
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Darwin) os="darwin" ;;
  Linux)  os="linux" ;;
  *) echo "unsupported OS: $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64)  arch="amd64" ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# --- Resolve version -----------------------------------------------------
if [ -z "$VERSION" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')"
fi
if [ -z "$VERSION" ]; then
  echo "could not determine latest version" >&2
  exit 1
fi
echo "Installing ${BINARY} ${VERSION} for ${os}/${arch}"

# --- Download ------------------------------------------------------------
BASE="https://github.com/${REPO}/releases/download/${VERSION}"
ASSET="${BINARY}_${VERSION#v}_${os}_${arch}.tar.gz"
CHECKSUMS="${BINARY}_${VERSION#v}_SHA256SUMS"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "  downloading ${ASSET}"
curl -fsSL -o "${TMPDIR}/${ASSET}" "${BASE}/${ASSET}"
curl -fsSL -o "${TMPDIR}/${CHECKSUMS}" "${BASE}/${CHECKSUMS}"

# --- Verify checksum -----------------------------------------------------
echo "  verifying checksum"
(
  cd "$TMPDIR"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum --ignore-missing --quiet --check "$CHECKSUMS"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 --ignore-missing --quiet --check "$CHECKSUMS"
  else
    echo "no sha256 tool available (install coreutils or use shasum)" >&2
    exit 1
  fi
)

# --- Extract and install -------------------------------------------------
echo "  extracting"
tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR"

mkdir -p "$INSTALL_DIR"
install -m 0755 "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

echo "✓ installed ${BINARY} ${VERSION} → ${INSTALL_DIR}/${BINARY}"

# --- PATH hint -----------------------------------------------------------
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Add ${INSTALL_DIR} to your PATH:"
    echo "  echo 'export PATH=\"\$PATH:${INSTALL_DIR}\"' >> ~/.zshrc"
    echo "  source ~/.zshrc"
    ;;
esac