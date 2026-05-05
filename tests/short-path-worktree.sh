#!/usr/bin/env bash
# Regression test: tmux-short-path correctly names worktrees even when cwd
# is a subdirectory inside the worktree (not just the worktree root).
#
# Run:  bash tests/short-path-worktree.sh
# Integrate into bin/tmux-claude-test once PR #1 (agent/audit-test-infra) lands.

set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$REPO/bin/tmux-short-path"

pass=0; fail=0

PASS() { pass=$((pass + 1)); printf "  PASS  %s\n" "$1"; }
FAIL() { fail=$((fail + 1)); printf "  FAIL  %s — %s\n" "$1" "${2:-}"; }

ZSH_BIN=$(command -v zsh 2>/dev/null || true)
if [ -z "$ZSH_BIN" ]; then
  echo "SKIP  zsh not available"
  exit 0
fi

run() { HOME=/tmp "$ZSH_BIN" "$SCRIPT" "$1" 2>/dev/null; }

echo "── tmux-short-path worktree regression ──"
echo

# Root of a worktree — always worked; must stay correct
out=$(run "/tmp/.worktrees/feature-x")
if [ "$out" = "wt:feature-x" ]; then
  PASS "worktree root → wt:feature-x"
else
  FAIL "worktree root → wt:feature-x" "got: '$out'"
fi

# Subdirectory inside a worktree — was broken (showed last dir component)
out=$(run "/tmp/.worktrees/feature-x/src/components")
if [ "$out" = "wt:feature-x" ]; then
  PASS "worktree subdir → wt:feature-x (not 'wt:components')"
else
  FAIL "worktree subdir → wt:feature-x" "got: '$out' (old bug returned 'wt:components')"
fi

# Multiple levels deep
out=$(run "/tmp/.worktrees/my-branch/a/b/c/d")
if [ "$out" = "wt:my-branch" ]; then
  PASS "worktree deep subdir → wt:my-branch"
else
  FAIL "worktree deep subdir → wt:my-branch" "got: '$out'"
fi

# Long worktree name — truncation still works for root path
long="averylongworktreenamethatexceedslimit"
out=$(run "/tmp/.worktrees/$long")
if printf '%s' "$out" | grep -q '^wt:' && [ "${#out}" -le 22 ]; then
  PASS "long worktree name truncated (${#out} chars)"
else
  FAIL "long worktree name truncated" "got (${#out} chars): '$out'"
fi

# Long worktree name — truncation works from nested path too
out=$(run "/tmp/.worktrees/$long/src/foo")
if printf '%s' "$out" | grep -q '^wt:' && [ "${#out}" -le 22 ]; then
  PASS "long worktree name truncated from nested path (${#out} chars)"
else
  FAIL "long worktree name truncated from nested path" "got (${#out} chars): '$out'"
fi

echo
printf "── Result ──\n  %d PASS / %d FAIL\n\n" "$pass" "$fail"
[ "$fail" -gt 0 ] && exit 1
exit 0
