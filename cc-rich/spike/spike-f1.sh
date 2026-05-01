#!/usr/bin/env bash
# cc-rich/spike/spike-f1.sh — verify whether `claude --resume <new-sid>`
# accepts a JSONL prefix-copy of an existing session as a fresh fork.
#
# Outputs PASS or FAIL with diagnostic info. Throwaway — kept in repo
# only so the result can be re-verified on a different machine.

set -uo pipefail

PROJECTS="$HOME/.claude/projects"

# 1. Find any existing session jsonl with at least 4 lines
src=""
while read -r f; do
  n=$(wc -l < "$f" 2>/dev/null | tr -d ' ')
  if [ "$n" -ge 4 ]; then
    src="$f"
    break
  fi
done < <(find "$PROJECTS" -maxdepth 3 -name '*.jsonl' -type f 2>/dev/null)

if [ -z "$src" ]; then
  echo "FAIL: no source session with >=4 lines found in $PROJECTS" >&2
  exit 1
fi

# 2. Generate a new UUID; copy first 4 lines to <project>/<new-sid>.jsonl
new_sid=$(uuidgen | tr '[:upper:]' '[:lower:]')
proj_dir=$(dirname "$src")
new_path="$proj_dir/$new_sid.jsonl"
head -4 "$src" > "$new_path"

echo "spike: src=$src" >&2
echo "spike: new=$new_path" >&2
echo "spike: new_sid=$new_sid" >&2

# 3. Try to resume. Capture exit code and first 3 lines of stdout/stderr.
out=$(claude --resume "$new_sid" --print "ping" 2>&1 < /dev/null | head -3)
code=$?

# 4. Cleanup the synthesized file regardless of outcome
rm -f "$new_path"

# 5. Report
if [ $code -eq 0 ]; then
  echo "PASS: claude --resume accepted synthesized sid"
  echo "  output: $out"
  exit 0
else
  echo "FAIL: claude --resume rejected synthesized sid (exit $code)"
  echo "  output: $out"
  exit 2
fi

# Result on AMACY10J4KK76 @ 2026-05-01:
#   FAIL (exit 1)
#   claude --resume rejected synthesized sid: "No conversation found with session ID: aa89856b-0afc-4a96-86bd-eb63fabb6a8e"
#   Conclusion: Claude validates session IDs server-side (or against an internal index); a JSONL
#   prefix-copy placed in the correct project dir is NOT sufficient to forge a resumable session.
#   F1 must fall back to F2-with-preamble.
