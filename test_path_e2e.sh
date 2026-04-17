#!/bin/sh
# PATH e2e tests — verify that install.sh configure_path() writes the correct
# shell rc file and that a fresh shell subprocess with a scrubbed environment
# finds driftr's bin dir on PATH automatically.
#
# Prerequisites:
#   - driftr binary must be on PATH (set by caller)
#   - install.sh must exist in the current directory
#
# Environment:
#   TEST_SHELL  zsh | bash | fish | all (default: all)
#
# Run inside Dockerfile.path-e2e (Linux) or natively on macOS after:
#   export PATH="/path/to/driftr-binary-dir:$PATH"
#   sh test_path_e2e.sh

set -eu

pass=0
fail_count=0

ok() {
    printf '[PASS] %s\n' "$1"
    pass=$((pass + 1))
}

fail() {
    printf '[FAIL] %s\n' "$1"
    fail_count=$((fail_count + 1))
}

assert_cmd() {
    desc="$1"; shift
    if "$@" > /dev/null 2>&1; then
        ok "$desc"
    else
        fail "$desc"
    fi
}

assert_fail() {
    desc="$1"; shift
    if "$@" > /dev/null 2>&1; then
        fail "$desc (expected failure, got success)"
    else
        ok "$desc"
    fi
}

# ── setup ────────────────────────────────────────────────────────────────────

INSTALL_DIR="$HOME/.driftr/bin"

# Scrubbed PATH: excludes /usr/local/bin where the driftr binary lives in Docker.
# Every PATH assertion must find the sentinel via the rc file only.
CLEAN_PATH="/usr/bin:/usr/sbin:/bin:/sbin"

printf 'driftr PATH e2e — shell: %s\n\n' "${TEST_SHELL:-all}"

# Create ~/.driftr/bin and shims.
driftr setup

# Sentinel: a unique executable in ~/.driftr/bin that won't exist elsewhere.
# Asserting 'command -v driftr-path-e2e-ok' proves the bin dir is on PATH via
# the rc file — not via /usr/local/bin where the driftr binary lives in Docker.
SENTINEL="driftr-path-e2e-ok"
printf '#!/bin/sh\necho ok\n' > "$INSTALL_DIR/$SENTINEL"
chmod +x "$INSTALL_DIR/$SENTINEL"

# Patched install.sh: disable the 'main' call at the last line so we can source
# just the function definitions and call configure_path() directly.
# install.sh defines functions then calls main on line 240 as a bare word;
# 'main() { :; }' before sourcing does not work because install.sh redefines
# main() after our definition. sed-patch is the reliable approach.
PATCHED_INSTALL=/tmp/install-path-test.sh
sed 's/^main$/: # disabled by test/' install.sh > "$PATCHED_INSTALL"

# Invoke configure_path for a given shell. Sets SHELL so detect_shell picks the
# right rc file target, then sources the patched install.sh and calls the function.
configure_path_for() {
    shell_bin="$1"
    (
        export INSTALL_DIR
        SHELL="$shell_bin"
        export SHELL
        # shellcheck source=install.sh
        . "$PATCHED_INSTALL"
        configure_path
    )
}

# ── zsh ──────────────────────────────────────────────────────────────────────

run_zsh_tests() {
    ZSH_BIN=$(command -v zsh 2>/dev/null) || { printf '[SKIP] zsh not installed\n'; return; }
    printf '\n[zsh]\n'

    configure_path_for "$ZSH_BIN"

    assert_cmd "zsh: .zshenv written" \
        test -f "$HOME/.zshenv"
    assert_cmd "zsh: .zshenv contains driftr bin dir" \
        grep -qF "$INSTALL_DIR" "$HOME/.zshenv"

    # No --no-rcs: tests that zsh auto-sources .zshenv on every invocation.
    # That auto-sourcing behaviour is why .zshenv is the correct target.
    assert_cmd "zsh: non-interactive invocation sources .zshenv" \
        env -i HOME="$HOME" PATH="$CLEAN_PATH" \
        zsh -c "command -v $SENTINEL"
    assert_cmd "zsh: interactive invocation sources .zshenv" \
        env -i HOME="$HOME" PATH="$CLEAN_PATH" \
        zsh -i -c "command -v $SENTINEL"
}

# ── bash ─────────────────────────────────────────────────────────────────────

run_bash_tests() {
    BASH_BIN=$(command -v bash 2>/dev/null) || { printf '[SKIP] bash not installed\n'; return; }
    printf '\n[bash]\n'

    configure_path_for "$BASH_BIN"

    assert_cmd "bash: .bash_profile written" \
        test -f "$HOME/.bash_profile"
    assert_cmd "bash: .bash_profile contains driftr bin dir" \
        grep -qF "$INSTALL_DIR" "$HOME/.bash_profile"

    assert_cmd "bash: login shell resolves driftr" \
        env -i HOME="$HOME" PATH="$CLEAN_PATH" \
        bash -l -c "command -v $SENTINEL"

    # Non-interactive non-login bash does NOT read .bash_profile — documented
    # limitation. Asserting the failure makes the contract executable and
    # prevents accidental "fixes" that paper over the boundary.
    assert_fail "bash: non-interactive non-login does not resolve driftr (expected)" \
        env -i HOME="$HOME" PATH="$CLEAN_PATH" \
        bash -c "command -v $SENTINEL"

    # Non-interactive children of a login shell inherit PATH.
    assert_cmd "bash: child of login shell inherits driftr on PATH" \
        env -i HOME="$HOME" PATH="$CLEAN_PATH" \
        bash -l -c "bash -c 'command -v $SENTINEL'"
}

# ── fish ─────────────────────────────────────────────────────────────────────

run_fish_tests() {
    FISH_BIN=$(command -v fish 2>/dev/null) || { printf '[SKIP] fish not installed\n'; return; }
    printf '\n[fish]\n'

    configure_path_for "$FISH_BIN"

    FISH_CONF="${XDG_CONFIG_HOME:-$HOME/.config}/fish/conf.d/driftr.fish"
    assert_cmd "fish: conf.d/driftr.fish written" \
        test -f "$FISH_CONF"
    assert_cmd "fish: driftr.fish contains set -gx PATH" \
        grep -q 'set -gx PATH' "$FISH_CONF"

    # fish sources conf.d on every invocation, interactive or not.
    assert_cmd "fish: non-interactive invocation resolves driftr" \
        env -i HOME="$HOME" PATH="$CLEAN_PATH" \
        fish -c "command -v $SENTINEL"
    assert_cmd "fish: type -q succeeds" \
        env -i HOME="$HOME" PATH="$CLEAN_PATH" \
        fish -c "type -q $SENTINEL"
}

# ── dispatch ─────────────────────────────────────────────────────────────────

case "${TEST_SHELL:-all}" in
    zsh)  run_zsh_tests ;;
    bash) run_bash_tests ;;
    fish) run_fish_tests ;;
    all)
        run_zsh_tests
        run_bash_tests
        run_fish_tests
        ;;
    *)
        printf 'error: unknown TEST_SHELL=%s\n' "$TEST_SHELL" >&2
        exit 1
        ;;
esac

# ── summary ──────────────────────────────────────────────────────────────────

printf '\n========================================\n'
printf 'PATH e2e: %d passed, %d failed\n' "$pass" "$fail_count"
printf '========================================\n'
[ "$fail_count" -eq 0 ]
