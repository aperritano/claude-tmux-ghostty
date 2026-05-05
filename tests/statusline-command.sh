#!/usr/bin/env bash
# tests/statusline-command.sh — regression tests for claude/statusline-command.sh
#
# The statusline script is the interface between Claude Code and the tmux
# status bar. It parses JSON from stdin and renders an ANSI status line.
# Tests here verify all JSON field paths, traffic-light color thresholds,
# and graceful handling of missing fields.
#
# Integration note: once any of the open audit/test PRs land, these tests
# should be merged into bin/tmux-claude-test under a "statusline-command.sh"
# section. Until then, run this standalone:
#
#   bash tests/statusline-command.sh
#
# Requires: bash, jq (for building test JSON only), no tmux or zsh needed.
# Exit 0 = all pass; 1 = any fail.

set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
SL="$REPO/claude/statusline-command.sh"

PASS=0; FAIL=0; SKIP=0

pass() { PASS=$((PASS+1)); printf 'PASS  %s\n' "$1"; }
fail() { FAIL=$((FAIL+1)); printf 'FAIL  %s\n' "$1"; }
skip() { SKIP=$((SKIP+1)); printf 'SKIP  %s\n' "$1"; }

expect() {
  local got="$1" want="$2" desc="$3"
  if [ "$got" = "$want" ]; then
    pass "$desc"
  else
    fail "$desc  (want: $(printf '%q' "$want")  got: $(printf '%q' "$got"))"
  fi
}

expect_contains() {
  local got="$1" substr="$2" desc="$3"
  if printf '%s' "$got" | grep -qF "$substr"; then
    pass "$desc"
  else
    fail "$desc  (want substring: $(printf '%q' "$substr")  got: $(printf '%q' "$got"))"
  fi
}

expect_not_contains() {
  local got="$1" substr="$2" desc="$3"
  if printf '%s' "$got" | grep -qF "$substr"; then
    fail "$desc  (unexpected: $(printf '%q' "$substr")  in: $(printf '%q' "$got"))"
  else
    pass "$desc"
  fi
}

# Run the statusline script with a JSON string on stdin
run_sl() { printf '%s' "$1" | bash "$SL" 2>/dev/null; }

# ── Prerequisite: script exists and is executable ─────────
if [ ! -x "$SL" ]; then
  printf 'FATAL: %s is missing or not executable\n' "$SL"
  exit 1
fi

printf '# tests/statusline-command.sh\n\n'

# ── Full JSON: all fields present ─────────────────────────
printf '## Full JSON output\n'

FULL='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"claude-opus-4"},"context_window":{"used_percentage":42},"session_name":"myproject"}'
OUT=$(run_sl "$FULL")

expect_contains "$OUT" "claude-opus-4"   "full JSON: model name in output"
expect_contains "$OUT" "myproject"       "full JSON: session_name in output"
expect_contains "$OUT" "ctx:42%"         "full JSON: context percentage in output"
expect_contains "$OUT" "/tmp"            "full JSON: directory in output"

# user@host should always appear
HOST=$(hostname -s 2>/dev/null || true)
USER_NAME=$(whoami 2>/dev/null || true)
if [ -n "$USER_NAME" ] && [ -n "$HOST" ]; then
  expect_contains "$OUT" "${USER_NAME}@${HOST}" "full JSON: user@host in output"
fi

# ── Home directory abbreviation ───────────────────────────
printf '\n## Home directory abbreviation\n'

JSON="{\"workspace\":{\"current_dir\":\"$HOME\"},\"model\":{\"display_name\":\"x\"}}"
OUT=$(run_sl "$JSON")
expect_contains "$OUT" "~" "home dir: abbreviated to ~"
expect_not_contains "$OUT" "$HOME/" "home dir: full path not shown"

JSON="{\"workspace\":{\"current_dir\":\"$HOME/projects/foo\"},\"model\":{\"display_name\":\"x\"}}"
OUT=$(run_sl "$JSON")
expect_contains "$OUT" "~/projects/foo" "home subdir: abbreviated correctly"

# ── .cwd fallback key ─────────────────────────────────────
printf '\n## .cwd fallback (legacy key)\n'

JSON='{"cwd":"/srv","model":{"display_name":"fallback-test"}}'
OUT=$(run_sl "$JSON")
expect_contains "$OUT" "fallback-test" ".cwd fallback: model in output"
expect_contains "$OUT" "/srv"          ".cwd fallback: directory in output"

# ── Missing optional fields ───────────────────────────────
printf '\n## Missing optional fields\n'

# No session_name → no session label (no text that looks like a bracketed word)
JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":42}}'
OUT=$(run_sl "$JSON")
expect_not_contains "$OUT" "myproject" "missing session_name: label absent"

# No context_window → no ctx: segment
JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"}}'
OUT=$(run_sl "$JSON")
expect_not_contains "$OUT" "ctx:" "missing context_window: ctx absent"

# No model → model shows as "?" (jq fallback)
JSON='{"workspace":{"current_dir":"/tmp"}}'
OUT=$(run_sl "$JSON")
expect_contains "$OUT" "?" "missing model: shows ? fallback"

# Minimal valid JSON (just workspace): must not crash
JSON='{"workspace":{"current_dir":"/tmp"}}'
OUT=$(run_sl "$JSON")
if [ -n "$OUT" ]; then
  pass "minimal JSON: non-empty output (no crash)"
else
  fail "minimal JSON: empty output (possible crash)"
fi

# ── Context traffic-light color thresholds ────────────────
printf '\n## Traffic-light color thresholds (ANSI codes)\n'

# ESC character for case patterns
ESC=$'\033'

# ctx < 50 → green (\033[32m)
JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":0}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[32m"*) pass "ctx 0%: green color" ;;
  *) fail "ctx 0%: expected green (ESC[32m), got: $(printf '%q' "$OUT")" ;;
esac

JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":30}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[32m"*) pass "ctx 30%: green color" ;;
  *) fail "ctx 30%: expected green (ESC[32m)" ;;
esac

JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":49}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[32m"*) pass "ctx 49%: green color (below 50 threshold)" ;;
  *) fail "ctx 49%: expected green (ESC[32m)" ;;
esac

# ctx >= 50, < 80 → yellow (\033[33m)
JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":50}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[33m"*) pass "ctx 50%: yellow color (at 50 threshold)" ;;
  *) fail "ctx 50%: expected yellow (ESC[33m)" ;;
esac

JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":60}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[33m"*) pass "ctx 60%: yellow color" ;;
  *) fail "ctx 60%: expected yellow (ESC[33m)" ;;
esac

JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":79}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[33m"*) pass "ctx 79%: yellow color (below 80 threshold)" ;;
  *) fail "ctx 79%: expected yellow (ESC[33m)" ;;
esac

# ctx >= 80 → red (\033[31m)
JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":80}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[31m"*) pass "ctx 80%: red color (at 80 threshold)" ;;
  *) fail "ctx 80%: expected red (ESC[31m)" ;;
esac

JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":85}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[31m"*) pass "ctx 85%: red color" ;;
  *) fail "ctx 85%: expected red (ESC[31m)" ;;
esac

JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"},"context_window":{"used_percentage":100}}'
OUT=$(run_sl "$JSON")
case "$OUT" in
  *"${ESC}[31m"*) pass "ctx 100%: red color" ;;
  *) fail "ctx 100%: expected red (ESC[31m)" ;;
esac

# ── Git info segment ──────────────────────────────────────
printf '\n## Git info segment\n'

# In a git repo: output should include repo name and branch info
JSON="{\"workspace\":{\"current_dir\":\"$REPO\"},\"model\":{\"display_name\":\"x\"}}"
OUT=$(run_sl "$JSON")
expect_contains "$OUT" "claude-tmux-ghostty" "git repo: repo name in output"

# In /tmp (no git): no git segment
JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"}}'
OUT=$(run_sl "$JSON")
expect_not_contains "$OUT" ":(" "non-git dir: no branch segment"

# ── Output format: no trailing newline ────────────────────
printf '\n## Output format\n'

JSON='{"workspace":{"current_dir":"/tmp"},"model":{"display_name":"x"}}'
RAW=$(run_sl "$JSON"; echo x)  # append 'x' so trailing newline isn't swallowed
if [ "${RAW%x}" = "$(run_sl "$JSON")" ]; then
  pass "output: no trailing newline (uses printf not echo)"
else
  fail "output: unexpected trailing newline"
fi

# ── Summary ───────────────────────────────────────────────
printf '\n%d PASS / %d SKIP / %d FAIL\n' "$PASS" "$SKIP" "$FAIL"
[ "$FAIL" -eq 0 ]
