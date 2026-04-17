#!/bin/sh
# Install shell packages — abstracts apt-get (Debian) and apk (Alpine).
# Usage: sh install-deps.sh "zsh fish bash"
set -eu
pkgs="$1"
if command -v apt-get > /dev/null 2>&1; then
    apt-get update && apt-get install -y --no-install-recommends ca-certificates curl $pkgs
    rm -rf /var/lib/apt/lists/*
elif command -v apk > /dev/null 2>&1; then
    apk add --no-cache ca-certificates curl $pkgs
else
    echo "error: no supported package manager found" >&2
    exit 1
fi
