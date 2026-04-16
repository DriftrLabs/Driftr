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
    case ":${PATH:-}:" in
        *":${INSTALL_DIR}:"*)
            log "Run 'driftr --version' to verify."
            ;;
        *)
            log "Open a new shell, or run this in the current shell:"
            log "  export PATH=\"${INSTALL_DIR}:\$PATH\""
            ;;
    esac
    log ""
    log "To enable shell completions:"
    log "  # zsh"
    log "  echo 'eval \"\$(driftr completion zsh)\"' >> ~/.zshrc"
    log "  # bash"
    log "  echo 'eval \"\$(driftr completion bash)\"' >> ~/.bashrc"
    log "  # fish"
    log "  driftr completion fish > ~/.config/fish/completions/driftr.fish"
}

configure_path() {
    # Skip if the install dir is already on PATH in the current environment.
    case ":${PATH:-}:" in
        *":${INSTALL_DIR}:"*)
            log "${INSTALL_DIR} already on PATH"
            return
            ;;
    esac

    shell_name="$(basename "${SHELL:-/bin/sh}")"

    # Fish uses different syntax and a different config location.
    if [ "$shell_name" = "fish" ]; then
        fish_conf_dir="${XDG_CONFIG_HOME:-$HOME/.config}/fish/conf.d"
        fish_profile="${fish_conf_dir}/driftr.fish"
        if [ -f "$fish_profile" ] && grep -qF "$INSTALL_DIR" "$fish_profile"; then
            return
        fi
        mkdir -p "$fish_conf_dir"
        printf '# Driftr\nset -gx PATH %s $PATH\n' "$INSTALL_DIR" >> "$fish_profile"
        log "Added ${INSTALL_DIR} to PATH in ${fish_profile}"
        return
    fi

    path_line="export PATH=\"${INSTALL_DIR}:\$PATH\""

    # Pick the profile that is read by ALL invocations of the shell (not just
    # interactive ones) so driftr works in scripts, IDE terminals, and cron.
    #   - zsh:  .zshenv is sourced on every invocation
    #   - bash: .bash_profile for login shells, .bashrc for interactive;
    #           also write to .profile as a POSIX fallback for non-login
    #           non-interactive shells that source it
    case "$shell_name" in
        zsh)
            profile="${ZDOTDIR:-$HOME}/.zshenv"
            ;;
        bash)
            if [ -f "$HOME/.bash_profile" ] || [ ! -f "$HOME/.bashrc" ]; then
                profile="$HOME/.bash_profile"
            else
                profile="$HOME/.bashrc"
            fi
            ;;
        *)
            profile="$HOME/.profile"
            ;;
    esac

    # Dedup: only check the target file. If the user has the line in another
    # rc file from a prior install, warn but still write to the new target so
    # non-interactive shells pick it up.
    if [ -f "$profile" ] && grep -qF "$INSTALL_DIR" "$profile"; then
        log "${INSTALL_DIR} already configured in ${profile}"
        return
    fi

    for stale in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile" "$HOME/.bash_profile"; do
        [ "$stale" = "$profile" ] && continue
        if [ -f "$stale" ] && grep -qF "$INSTALL_DIR" "$stale"; then
            log "note: ${INSTALL_DIR} is also in ${stale} — consider removing that entry to avoid duplicate PATH"
        fi
    done

    printf '\n# Driftr\n%s\n' "$path_line" >> "$profile"
    log "Added ${INSTALL_DIR} to PATH in ${profile}"
}

main
