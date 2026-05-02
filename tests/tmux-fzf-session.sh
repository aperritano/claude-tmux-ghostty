#!/usr/bin/env bash
# Regression tests for bin/tmux-fzf-session.
# Runs without a live tmux server or fzf — those paths are exercised at launch.
# Exit 0 = all pass; Exit 1 = any fail.

set -uo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
script="$REPO/bin/tmux-fzf-session"
tmux_conf="$REPO/tmux/tmux.conf"

PASS=0; FAIL=0; SKIP=0
pass() { PASS=$((PASS+1)); printf 'PASS  %s\n' "$*"; }
fail() { FAIL=$((FAIL+1)); printf 'FAIL  %s\n' "$*"; }
skip() { SKIP=$((SKIP+1)); printf 'SKIP  %s\n' "$*"; }

# ── Script presence and permissions ──────────────────────────────────────────
[ -f "$script" ]  && pass "tmux-fzf-session: exists"      || fail "tmux-fzf-session: missing"
[ -x "$script" ]  && pass "tmux-fzf-session: executable"   || fail "tmux-fzf-session: not executable"

# ── Syntax check ─────────────────────────────────────────────────────────────
if bash -n "$script" 2>/dev/null; then
  pass "tmux-fzf-session: bash -n syntax clean"
else
  fail "tmux-fzf-session: bash -n syntax errors"
fi

# ── Guard: exits 1 immediately when not inside tmux ──────────────────────────
if [ -z "${TMUX:-}" ]; then
  "$script" 2>/dev/null; code=$?
  [ "$code" -eq 1 ] && pass "tmux-fzf-session: exits 1 outside tmux" \
                     || fail "tmux-fzf-session: expected exit 1 outside tmux, got $code"
else
  skip "tmux-fzf-session: running inside tmux — 'not in tmux' guard test skipped"
fi

# ── tmux.conf wiring ─────────────────────────────────────────────────────────
if grep -q 'bind S.*tmux-fzf-session' "$tmux_conf" 2>/dev/null; then
  pass "tmux.conf: bind S calls tmux-fzf-session"
else
  fail "tmux.conf: bind S for tmux-fzf-session not found"
fi

if grep -q 'Ctrl-a S' "$tmux_conf" 2>/dev/null; then
  pass "tmux.conf: Ctrl-a S documented in help popup"
else
  fail "tmux.conf: Ctrl-a S missing from help popup"
fi

# ── Summary ───────────────────────────────────────────────────────────────────
printf '\n%d PASS / %d SKIP / %d FAIL\n' "$PASS" "$SKIP" "$FAIL"
[ "$FAIL" -eq 0 ]
