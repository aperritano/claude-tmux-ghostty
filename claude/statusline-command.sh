#!/bin/sh
# Claude Code status line command — mirrors Powerlevel10k lean prompt style
# Segments: user@host  dir  repo:(branch) ↑↓  model  [session]  ctx:%
# Reads JSON from stdin and outputs a formatted status line

input=$(cat)

cwd=$(echo "$input" | jq -r '.workspace.current_dir // .cwd // "?"')
model=$(echo "$input" | jq -r '.model.display_name // "?"')
model_id=$(echo "$input" | jq -r '.model.id // empty')
transcript=$(echo "$input" | jq -r '.transcript_path // empty')
session=$(echo "$input" | jq -r '.session_name // empty')

# Compute context usage from transcript (Claude Code does not send a percentage field).
# Sum input + cache_creation + cache_read on the most recent assistant turn.
used_int=""
if [ -n "$transcript" ] && [ -r "$transcript" ]; then
  tokens=$(tail -r "$transcript" 2>/dev/null | head -200 | \
    jq -r 'select(.message.usage) | .message.usage
           | (.input_tokens // 0)
           + (.cache_creation_input_tokens // 0)
           + (.cache_read_input_tokens // 0)' 2>/dev/null | head -1)
  if [ -n "$tokens" ] && [ "$tokens" -gt 0 ] 2>/dev/null; then
    case "$model_id" in
      *'[1m]'*) ctx_max=1000000 ;;
      *)        ctx_max=200000  ;;
    esac
    used_int=$(( tokens * 100 / ctx_max ))
  fi
fi

# user@host
user=$(whoami)
host=$(hostname -s)

# Shorten home directory to ~
short_cwd=$(echo "$cwd" | sed "s|^$HOME|~|")

# Git branch + worktree + ahead/behind
branch=""
repo_label=""
sync=""
if git -C "$cwd" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  branch=$(GIT_OPTIONAL_LOCKS=0 git -C "$cwd" -c core.hooksPath=/dev/null symbolic-ref --short HEAD 2>/dev/null \
    || GIT_OPTIONAL_LOCKS=0 git -C "$cwd" rev-parse --short HEAD 2>/dev/null)

  # Detect worktree vs normal repo
  git_dir=$(git -C "$cwd" rev-parse --git-dir 2>/dev/null)
  common_dir=$(git -C "$cwd" rev-parse --git-common-dir 2>/dev/null)
  if [ "$git_dir" != "$common_dir" ]; then
    wt_name=$(basename "$git_dir")
    repo_label=".wt/${wt_name}"
  else
    toplevel=$(git -C "$cwd" rev-parse --show-toplevel 2>/dev/null)
    repo_label=$(basename "$toplevel")
  fi

  # Ahead/behind upstream
  counts=$(GIT_OPTIONAL_LOCKS=0 git -C "$cwd" rev-list --left-right --count HEAD...@{upstream} 2>/dev/null)
  if [ -n "$counts" ]; then
    ahead=$(echo "$counts" | cut -f1)
    behind=$(echo "$counts" | cut -f2)
    [ "$ahead" -gt 0 ] 2>/dev/null && sync="${sync}↑${ahead}"
    [ "$behind" -gt 0 ] 2>/dev/null && sync="${sync}↓${behind}"
  fi
fi

# user@host
line=$(printf '\033[32m%s@%s\033[0m' "$user" "$host")

# Directory
line="${line} $(printf '\033[34m%s\033[0m' "$short_cwd")"

# Repo + branch + sync
if [ -n "$branch" ]; then
  git_info="${repo_label}:(${branch})"
  [ -n "$sync" ] && git_info="${git_info} ${sync}"
  line="${line} $(printf '\033[33m%s\033[0m' "$git_info")"
fi

# Model
line="${line} $(printf '\033[36m%s\033[0m' "$model")"

# Session name if set
if [ -n "$session" ]; then
  line="${line} $(printf '\033[35m[%s]\033[0m' "$session")"
fi

# tmux-claude-save freshness — visible signal that continuum auto-save is alive.
# Green: <6m (healthy)  Yellow: 6-15m (overdue)  Red: >15m (continuum may be dead)
if [ -L "$HOME/.tmux/resurrect/last" ]; then
  save_resolved=$(readlink "$HOME/.tmux/resurrect/last")
  save_path="$HOME/.tmux/resurrect/$save_resolved"
  if [ -f "$save_path" ]; then
    save_mtime=$(stat -f %m "$save_path" 2>/dev/null || echo 0)
    save_age_sec=$(( $(date +%s) - save_mtime ))
    if   [ "$save_age_sec" -lt 60   ]; then save_age="${save_age_sec}s"
    elif [ "$save_age_sec" -lt 3600 ]; then save_age="$((save_age_sec/60))m"
    else                                    save_age="$((save_age_sec/3600))h"; fi
    if   [ "$save_age_sec" -lt 360 ];  then save_color='\033[32m'
    elif [ "$save_age_sec" -lt 900 ];  then save_color='\033[33m'
    else                                    save_color='\033[31m'; fi
    line="${line} $(printf "${save_color}save:${save_age}\033[0m")"
  fi
fi

# Context usage with traffic-light coloring
if [ -n "$used_int" ]; then
  if [ "$used_int" -ge 80 ]; then
    color='\033[31m'
  elif [ "$used_int" -ge 50 ]; then
    color='\033[33m'
  else
    color='\033[32m'
  fi
  line="${line} $(printf "${color}ctx:${used_int}%%\033[0m")"
fi

printf '%s' "$line"
