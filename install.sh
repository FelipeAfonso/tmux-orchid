#!/usr/bin/env bash
# install.sh -- download and install a prebuilt tmux-orchid binary.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/FelipeAfonso/tmux-orchid/main/install.sh | bash
#
# Environment variables:
#   INSTALL_DIR  -- where to place the binary (default: ~/.local/bin)
#   VERSION      -- specific version to install (default: latest)

set -euo pipefail

REPO="FelipeAfonso/tmux-orchid"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# --- helpers ----------------------------------------------------------------

die() {
  printf "error: %s\n" "$1" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

# --- detect OS and architecture ---------------------------------------------

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)       die "unsupported operating system: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)             die "unsupported architecture: $(uname -m)" ;;
  esac
}

# --- resolve version --------------------------------------------------------

resolve_version() {
  if [ -n "${VERSION:-}" ]; then
    # Strip leading 'v' if present for consistency.
    echo "${VERSION#v}"
    return
  fi

  need curl

  local latest
  latest=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"v([^"]+)".*/\1/')

  [ -n "$latest" ] || die "could not determine latest release version"
  echo "$latest"
}

# --- download and verify ----------------------------------------------------

download_and_install() {
  local version="$1" os="$2" arch="$3"
  local name="tmux-orchid_${version}_${os}_${arch}"
  local tarball="${name}.tar.gz"
  local base_url="https://github.com/${REPO}/releases/download/v${version}"

  need curl
  need tar

  local tmpdir
  tmpdir=$(mktemp -d)
  trap 'rm -rf "$tmpdir"' EXIT

  printf "downloading %s ...\n" "$tarball"
  curl -fsSL "${base_url}/${tarball}" -o "${tmpdir}/${tarball}"
  curl -fsSL "${base_url}/checksums.txt" -o "${tmpdir}/checksums.txt"

  # Verify checksum.
  printf "verifying checksum ...\n"
  local expected
  expected=$(grep "${tarball}" "${tmpdir}/checksums.txt" | awk '{print $1}')
  [ -n "$expected" ] || die "checksum not found for ${tarball}"

  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "${tmpdir}/${tarball}" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "${tmpdir}/${tarball}" | awk '{print $1}')
  else
    die "no sha256sum or shasum found; cannot verify checksum"
  fi

  [ "$expected" = "$actual" ] || die "checksum mismatch: expected ${expected}, got ${actual}"

  # Extract and install.
  tar -xzf "${tmpdir}/${tarball}" -C "${tmpdir}"

  mkdir -p "${INSTALL_DIR}"
  install -m 755 "${tmpdir}/tmux-orchid" "${INSTALL_DIR}/tmux-orchid"

  printf "installed tmux-orchid v%s to %s/tmux-orchid\n" "$version" "$INSTALL_DIR"

  # PATH hint.
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
      printf "\nnote: %s is not in your PATH.\n" "$INSTALL_DIR"
      printf "add it with:\n\n  export PATH=\"%s:\$PATH\"\n\n" "$INSTALL_DIR"
      ;;
  esac
}

# --- main -------------------------------------------------------------------

main() {
  local os arch version
  os=$(detect_os)
  arch=$(detect_arch)
  version=$(resolve_version)

  printf "tmux-orchid v%s  %s/%s\n" "$version" "$os" "$arch"
  download_and_install "$version" "$os" "$arch"
}

main
