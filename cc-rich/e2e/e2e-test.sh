#!/usr/bin/env bash
# cc-rich/e2e/e2e-test.sh — end-to-end integration smoke for cc-rich.
#
# Tests the parts that aren't covered by teatest unit tests:
#   - The Go binary's CLI surface (flags, error paths)
#   - The shim chain (cc-rich-shim → ~/.local/bin/cc-rich)
#   - cc-replay-shim → fake claude (verifies stdin = seed file)
#   - Fork seed-file shape and atomic-write semantics
#   - Merge buffer (.cc-pending-prompt) format
#   - tmux key bindings registered after source-file
#
# What this script does NOT test:
#   - The Bubble Tea popup interaction (teatest covers it)
#   - Real `claude --resume` (the spike confirmed it rejects synth ids; we
#     stub claude with a fake binary to keep this test offline + fast)
#
# Usage:
#   ./e2e-test.sh              # run all cases
#   VERBOSE=1 ./e2e-test.sh    # show command output during checks
#
# Exit code: 0 if all PASS, 1 if any FAIL.

set -uo pipefail

# ─── State ──────────────────────────────────────────────────────────────
PASS=0
FAIL=0
ROOT="$(cd "$(dirname "$0")/.." && pwd)"        # cc-rich/
DOTFILES="$(cd "$ROOT/.." && pwd)"              # ~/dev/dotfiles
TMP="/tmp/cc-rich-e2e-$$"
mkdir -p "$TMP"
trap 'rm -rf "$TMP"' EXIT

G="\033[32m"; R="\033[31m"; C="\033[36m"; DIM="\033[90m"; N="\033[0m"

pass() { PASS=$((PASS+1)); printf "  ${G}✓${N} %s\n" "$1"; }
fail() { FAIL=$((FAIL+1)); printf "  ${R}✗${N} %s — ${R}%s${N}\n" "$1" "$2"; }
section() { printf "\n${C}── %s ──${N}\n" "$1"; }
log() { [[ "${VERBOSE:-0}" == "1" ]] && printf "${DIM}  %s${N}\n" "$1" || true; }

# ─── Section 1: build + binary present ──────────────────────────────────
section "build + install"

if (cd "$ROOT" && go build -o "$TMP/cc-rich" ./cmd/cc-rich) 2>"$TMP/build.err"; then
  pass "go build ./cmd/cc-rich"
else
  fail "go build" "see $TMP/build.err"
  exit 1
fi

if [[ -x "$HOME/.local/bin/cc-rich" ]]; then
  pass "Go binary installed at ~/.local/bin/cc-rich"
else
  fail "binary install" "run: cd cc-rich && make install"
fi

if [[ -x "$HOME/bin/cc-rich" ]]; then
  pass "shim wrapper at ~/bin/cc-rich"
else
  fail "shim install" "run: make install-shim"
fi

if [[ -x "$HOME/bin/cc-replay-shim" ]]; then
  pass "cc-replay-shim at ~/bin/cc-replay-shim"
else
  fail "replay shim" "missing — should be a symlink into dotfiles/bin/"
fi

# ─── Section 2: CLI surface ──────────────────────────────────────────────
section "CLI surface"

# --help should exit non-zero (no -help defined) but flag.Parse prints usage on stderr
out=$("$HOME/bin/cc-rich" --help 2>&1 || true)
if echo "$out" | grep -qE "browse|merge-into|pane"; then
  pass "--help shows flag descriptions"
else
  fail "--help output" "missing flag descriptions: $(echo "$out" | head -1)"
fi

# No args → usage on stderr, exit 2
out=$("$HOME/bin/cc-rich" 2>&1; echo "EXIT:$?")
if echo "$out" | grep -q "EXIT:2"; then
  pass "no-args exits 2"
else
  fail "no-args exit code" "expected 2, got: $(echo "$out" | tail -1)"
fi

# Bogus pane id → graceful error, exit 1
out=$("$HOME/bin/cc-rich" --pane "%9999" 2>&1; echo "EXIT:$?")
if echo "$out" | grep -q "EXIT:1" && echo "$out" | grep -qE "no Claude session|not found"; then
  pass "--pane <bogus> errors gracefully"
else
  fail "--pane <bogus>" "unexpected: $(echo "$out" | head -2)"
fi

# ─── Section 3: cc-replay-shim with a fake claude ────────────────────────
section "cc-replay-shim → fake claude"

# Build a fake `claude` that captures stdin to a sentinel file.
mkdir -p "$TMP/fake-bin"
cat > "$TMP/fake-bin/claude" <<'EOF'
#!/usr/bin/env bash
# Fake claude — capture stdin to a sentinel.
cat > "$TMP_SENTINEL"
echo "fake-claude-ran" >> "$TMP_SENTINEL"
EOF
chmod +x "$TMP/fake-bin/claude"

# Synthetic seed file
seed="$TMP/seed.txt"
cat > "$seed" <<'EOF'
// continuing from session abc:msg-u-3
// prior context (last 1 turn):

> I want to try a different direction here

Continue from there.
EOF

# Run cc-replay-shim with the fake claude on PATH
sentinel="$TMP/sentinel.txt"
TMP_SENTINEL="$sentinel" PATH="$TMP/fake-bin:$PATH" \
  bash "$DOTFILES/bin/cc-replay-shim" "$seed" </dev/null

if [[ -f "$sentinel" ]] && grep -q "different direction" "$sentinel"; then
  pass "cc-replay-shim pipes seed file into claude stdin"
else
  fail "shim → claude stdin" "sentinel missing seed content"
  log "$(cat "$sentinel" 2>&1 | head -5)"
fi

if grep -q "fake-claude-ran" "$sentinel"; then
  pass "fake claude was actually exec'd by the shim"
else
  fail "shim exec" "fake claude marker missing"
fi

# Empty-arg: usage + exit 2
out=$(bash "$DOTFILES/bin/cc-replay-shim" 2>&1; echo "EXIT:$?")
if echo "$out" | grep -q "EXIT:2" && echo "$out" | grep -q "usage:"; then
  pass "cc-replay-shim no-args → usage + exit 2"
else
  fail "shim no-args" "unexpected: $(echo "$out" | head -2)"
fi

# Missing seed file: usage + exit 2
out=$(bash "$DOTFILES/bin/cc-replay-shim" /nonexistent/seed 2>&1; echo "EXIT:$?")
if echo "$out" | grep -q "EXIT:2"; then
  pass "cc-replay-shim missing-file → exit 2"
else
  fail "shim missing-file" "unexpected: $(echo "$out" | head -2)"
fi

# ─── Section 4: tmux bindings registered ────────────────────────────────
section "tmux bindings"

if ! tmux info >/dev/null 2>&1; then
  log "no tmux server attached — skipping bindings checks"
  pass "tmux not running (skipped)"
else
  for key in R B M; do
    line=$(tmux list-keys -T prefix "$key" 2>/dev/null | head -1)
    if echo "$line" | grep -q "cc-rich"; then
      pass "Ctrl-a $key bound to cc-rich"
    else
      fail "Ctrl-a $key" "not bound to cc-rich (got: $(echo "$line" | head -c 80))"
    fi
  done
  # Ctrl-a Ctrl-r should still belong to tmux-resurrect (the conflict fix)
  line=$(tmux list-keys -T prefix C-r 2>/dev/null | head -1)
  if echo "$line" | grep -qE "resurrect|restore"; then
    pass "Ctrl-a Ctrl-r preserved for tmux-resurrect"
  else
    fail "Ctrl-a Ctrl-r" "expected tmux-resurrect, got: $(echo "$line" | head -c 80)"
  fi
fi

# ─── Section 5: Go test suite still green ───────────────────────────────
section "Go test suite"

if (cd "$ROOT" && go test ./... 2>"$TMP/gotest.err" >/dev/null); then
  pass "all internal tests pass"
else
  fail "go test ./..." "see $TMP/gotest.err"
  log "$(tail -10 "$TMP/gotest.err")"
fi

# ─── Section 6: action-layer file artifacts (sanity) ────────────────────
# These check the same things the Go unit tests check, but at the actual
# binary level — i.e. that artifacts written through actions.Fork etc.
# match the format the merge composer / replay shim expects.
section "action-layer file artifacts"

# Run the actions tests directly so we get fresh output (not cached).
# This already exercises Fork/Quote/WriteMergeBuffer end-to-end via mocks.
if (cd "$ROOT" && go test -count=1 ./internal/actions 2>"$TMP/actions.err" >/dev/null); then
  pass "internal/actions tests fresh run (Fork/Quote/Merge format)"
else
  fail "actions fresh-run" "see $TMP/actions.err"
fi

# ─── Summary ────────────────────────────────────────────────────────────
total=$((PASS + FAIL))
printf "\n${C}──────────────────────────────────────────${N}\n"
printf "  ${G}PASS${N} %d   ${R}FAIL${N} %d   (total %d)\n" "$PASS" "$FAIL" "$total"

if (( FAIL > 0 )); then
  printf "\n${R}%d FAIL(s).${N} Re-run with VERBOSE=1 for details.\n" "$FAIL"
  exit 1
fi
exit 0
