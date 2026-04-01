#!/bin/sh
# Driftr installer — downloads the latest release from GitHub and sets up PATH.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/DriftrLabs/driftr/main/install.sh | sh
#
# Environment variables:
#   DRIFTR_VERSION     — pin a specific version (e.g. "0.1.0"), default: latest
#   DRIFTR_INSTALL_DIR — override install directory, default: ~/.driftr/bin

set -eu

REPO="DriftrLabs/driftr"
DEFAULT_INSTALL_DIR="$HOME/.driftr/bin"
INSTALL_DIR="${DRIFTR_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"

# --- helpers ----------------------------------------------------------------

log() {
    printf '%s\n' "$@"
}

err() {
    log "error: $*" >&2
    exit 1
}

need() {
    if ! command -v "$1" >/dev/null 2>&1; then
        err "$1 is required but not found"
    fi
}

# Download a URL to a local file. Prefers curl, falls back to wget.
download() {
    url="$1"
    dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url"
    else
        err "curl or wget is required"
    fi
}

# Download a URL and print to stdout.
download_stdout() {
    url="$1"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url"
    else
        err "curl or wget is required"
    fi
}

# --- detect platform --------------------------------------------------------

detect_os() {
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       err "unsupported OS: $os" ;;
    esac
}

detect_arch() {
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             err "unsupported architecture: $arch" ;;
    esac
}

# --- resolve version --------------------------------------------------------

resolve_version() {
    if [ -n "${DRIFTR_VERSION:-}" ]; then
        # Strip leading "v" if present.
        echo "$DRIFTR_VERSION" | sed 's/^v//'
        return
    fi

    tag=$(download_stdout "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')

    if [ -z "$tag" ]; then
        err "could not determine latest version (GitHub API rate limit?)"
    fi

    echo "$tag"
}

# --- checksum verification --------------------------------------------------

verify_checksum() {
    archive="$1"
    checksums_file="$2"
    filename="$(basename "$archive")"

    expected=$(grep "$filename" "$checksums_file" | awk '{print $1}')
    if [ -z "$expected" ]; then
        err "no checksum found for $filename in checksums.txt"
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$archive" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$archive" | awk '{print $1}')
    else
        err "sha256sum or shasum is required for checksum verification"
    fi

    if [ "$actual" != "$expected" ]; then
        err "checksum mismatch for $filename (expected $expected, got $actual)"
    fi
}

# --- main -------------------------------------------------------------------

main() {
    need tar
    need uname

    os=$(detect_os)
    arch=$(detect_arch)
    version=$(resolve_version)

    archive_name="driftr_${version}_${os}_${arch}.tar.gz"
    base_url="https://github.com/${REPO}/releases/download/v${version}"

    log "Installing driftr v${version} (${os}/${arch})..."

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    log "Downloading ${archive_name}..."
    download "${base_url}/${archive_name}" "${tmpdir}/${archive_name}"
    download "${base_url}/checksums.txt" "${tmpdir}/checksums.txt"

    log "Verifying checksum..."
    verify_checksum "${tmpdir}/${archive_name}" "${tmpdir}/checksums.txt"

    log "Extracting..."
    tar -xzf "${tmpdir}/${archive_name}" -C "$tmpdir"

    mkdir -p "$INSTALL_DIR"
    cp "${tmpdir}/driftr" "${INSTALL_DIR}/driftr"
    chmod +x "${INSTALL_DIR}/driftr"

    log "Installed driftr to ${INSTALL_DIR}/driftr"

    # Run setup to create directories and generate shims.
    "${INSTALL_DIR}/driftr" setup

    # Configure PATH in shell profile if not already present.
    configure_path

    log ""
    log "driftr v${version} installed successfully!"
    log ""
    log "Restart your shell or run:"
    log "  export PATH=\"${INSTALL_DIR}:\$PATH\""
}

configure_path() {
    path_line="export PATH=\"${INSTALL_DIR}:\$PATH\""

    # Detect the user's shell profile.
    shell_name="$(basename "${SHELL:-/bin/sh}")"
    case "$shell_name" in
        zsh)  profile="$HOME/.zshrc" ;;
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                profile="$HOME/.bashrc"
            else
                profile="$HOME/.bash_profile"
            fi
            ;;
        *)    profile="$HOME/.profile" ;;
    esac

    # Don't duplicate the entry.
    if [ -f "$profile" ] && grep -qF "$INSTALL_DIR" "$profile"; then
        return
    fi

    printf '\n# Driftr\n%s\n' "$path_line" >> "$profile"
    log "Added ${INSTALL_DIR} to PATH in ${profile}"
}

main
