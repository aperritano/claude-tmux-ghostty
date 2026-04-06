#!/bin/sh
# Claude Code status line command — mirrors Powerlevel10k lean prompt style
# Segments: user@host  dir  repo:(branch) ↑↓  model  [session]  ctx:%
# Reads JSON from stdin and outputs a formatted status line

input=$(cat)

cwd=$(echo "$input" | jq -r '.workspace.current_dir // .cwd // "?"')
model=$(echo "$input" | jq -r '.model.display_name // "?"')
used=$(echo "$input" | jq -r '.context_window.used_percentage // empty')
session=$(echo "$input" | jq -r '.session_name // empty')

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

# Context usage with traffic-light coloring
if [ -n "$used" ]; then
  used_int=$(printf "%.0f" "$used")
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
