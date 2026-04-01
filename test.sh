#!/bin/bash

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

pass=0
fail=0

check() {
    local desc="$1"
    shift
    if "$@" > /dev/null 2>&1; then
        echo -e "  ${GREEN}✓${NC} $desc"
        pass=$((pass + 1))
    else
        echo -e "  ${RED}✗${NC} $desc"
        fail=$((fail + 1))
    fi
}

check_output() {
    local desc="$1"
    local expected="$2"
    shift 2
    local output
    output=$("$@" 2>&1) || true
    if echo "$output" | grep -q "$expected"; then
        echo -e "  ${GREEN}✓${NC} $desc"
        pass=$((pass + 1))
    else
        echo -e "  ${RED}✗${NC} $desc (expected '$expected', got '$output')"
        fail=$((fail + 1))
    fi
}

echo -e "${BLUE}═══════════════════════════════════════${NC}"
echo -e "${BLUE}  Driftr Integration Tests${NC}"
echo -e "${BLUE}═══════════════════════════════════════${NC}"
echo

# ── 1. Basic CLI ──────────────────────────────
echo -e "${BLUE}[1] Basic CLI${NC}"
check "driftr --help works" driftr --help
check_output "help shows available commands" "Available Commands" driftr --help
check_output "list shows no versions" "No node versions installed" driftr list
echo

# ── 2. Setup ──────────────────────────────────
echo -e "${BLUE}[2] Setup${NC}"
check "driftr setup creates dirs and shims" driftr setup
check "~/.driftr/bin exists" test -d "$HOME/.driftr/bin"
check "~/.driftr/tools/node exists" test -d "$HOME/.driftr/tools/node"
check "~/.driftr/config exists" test -d "$HOME/.driftr/config"
check "node shim was created" test -f "$HOME/.driftr/bin/node"
check "npm shim was created" test -f "$HOME/.driftr/bin/npm"
check "npx shim was created" test -f "$HOME/.driftr/bin/npx"
echo

# ── 3. Install (with checksum verification) ──
echo -e "${BLUE}[3] Install Node.js${NC}"
echo "  (installing node@22 — this may take a moment...)"
INSTALL_OUTPUT=$(driftr install node@22 -v 2>&1) || true
echo "$INSTALL_OUTPUT" | head -10
check "driftr install node@22 succeeds" echo "$INSTALL_OUTPUT"
check_output "checksum was verified" "Checksum verified OK" echo "$INSTALL_OUTPUT"
check_output "list shows installed version" "22." driftr list
echo

# ── 4. Default ────────────────────────────────
echo -e "${BLUE}[4] Set Global Default${NC}"

# Get the installed version dynamically
INSTALLED=$(driftr list 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
echo "  (installed version: $INSTALLED)"

check "driftr default node@$INSTALLED succeeds" driftr default "node@$INSTALLED"
check_output "list marks default with *" "*" driftr list
check_output "which resolves node" "$INSTALLED" driftr which node
check_output "which shows global source" "global default" driftr which node
echo

# ── 5. Pin ────────────────────────────────────
echo -e "${BLUE}[5] Project Pinning${NC}"
mkdir -p /tmp/test-project && cd /tmp/test-project

check "driftr pin node@$INSTALLED succeeds" driftr pin "node@$INSTALLED"
check ".driftr.toml was created" test -f /tmp/test-project/.driftr.toml
check_output ".driftr.toml contains version" "$INSTALLED" cat /tmp/test-project/.driftr.toml
check_output "which shows project source" "project config" driftr which node

cd /home/driftr
echo

# ── 6. Resolver Tracing ──────────────────────
echo -e "${BLUE}[6] Resolver Tracing${NC}"
TRACE_OUTPUT=$(driftr which node -v 2>&1) || true
check_output "verbose which shows resolution steps" "\\[resolve\\]" echo "$TRACE_OUTPUT"
check_output "verbose which shows step numbers" "Step" echo "$TRACE_OUTPUT"
echo

# ── 7. Run ────────────────────────────────────
echo -e "${BLUE}[7] Run Command${NC}"
export PATH="$HOME/.driftr/bin:$PATH"
check_output "driftr run -- node -v works" "v$INSTALLED" driftr run --node "$INSTALLED" -- node -v
echo

# ── 8. Reinstall (idempotency) ───────────────
echo -e "${BLUE}[8] Reinstall Idempotency${NC}"
check "reinstalling same version succeeds" driftr install "node@$INSTALLED"
check_output "still shows installed version" "$INSTALLED" driftr list
echo

# ── 9. Shim Execution ────────────────────────
echo -e "${BLUE}[9] Shim Execution${NC}"
check_output "node shim resolves correct version" "v$INSTALLED" "$HOME/.driftr/bin/node" -v
check_output "npm shim executes successfully" "." "$HOME/.driftr/bin/npm" -v
echo

# ── Summary ───────────────────────────────────
echo ""
echo -e "${BLUE}═══════════════════════════════════════${NC}"
total=$((pass + fail))
if [ "$fail" -eq 0 ]; then
    echo -e "  ${GREEN}All $total tests passed!${NC}"
else
    echo -e "  ${GREEN}$pass passed${NC}, ${RED}$fail failed${NC} out of $total"
fi
echo -e "${BLUE}═══════════════════════════════════════${NC}"

if [ "$fail" -gt 0 ]; then
    exit 1
fi
exit 0
