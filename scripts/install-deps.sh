#!/bin/sh
# Install shell packages — abstracts apt-get (Debian) and apk (Alpine).
# Usage: sh install-deps.sh "zsh fish bash"
set -eu
if [ "$#" -lt 1 ]; then
    printf 'usage: sh install-deps.sh "<pkg1> [pkg2 ...]"\n' >&2
    exit 1
fi
pkgs="$1"
if command -v apt-get > /dev/null 2>&1; then
    # libatomic1: required by Node.js binaries on ARM64 Debian
    apt-get update && apt-get install -y --no-install-recommends ca-certificates curl libatomic1 $pkgs
    rm -rf /var/lib/apt/lists/*
elif command -v apk > /dev/null 2>&1; then
    apk add --no-cache ca-certificates curl $pkgs
else
    printf 'error: no supported package manager found\n' >&2
    exit 1
fi
