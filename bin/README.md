# bin/ — Claude Code + tmux + Ghostty helper scripts

Helper executables that compose with the rest of the dotfiles to provide a
self-saving, search-friendly, agent-team-aware Claude Code workflow.

**Run `cc-audit`** to verify all of these are wired correctly (93+ checks).
**Run `cc-test`** for the regression suite (20+ tests).
**Run `cc-doctor`** for the composite health check (audit + test + auto-fix).
**Run `cc-help`** (zsh function in `~/.zshrc`) for an interactive cheat sheet.

## Quick reference

| Layer | Scripts |
|---|---|
| **Picker / search** | `claude-tmux`, `claude-sessions-list`, `claude-search`, `claude-tail` |
| **Worktree workflow** | `claude-worktrees`, `claude-prune-worktrees`, `claude-dirty` |
| **Save/restore stack** | `tmux-claude-state`, `tmux-claude-resurrect-restore`, `tmux-claude-prune` |
| **Observability** | `claude-today`, `claude-touched`, `claude-stats`, `claude-running` |
| **Archive** | `claude-export`, `claude-pin` |
| **Audit / test** | `tmux-claude-audit`, `tmux-claude-test` |
| **Statusline** | `tmux-window-cell`, `tmux-cell-daemon`, `tmux-fleet-count`, `tmux-save-dot`, `tmux-mute-indicator` |
| **Hooks** | `cc-sound` |

## Top-level workflow scripts

### `claude-tmux`
fzf-based picker over **all past Claude Code sessions** (`~/.claude/projects/*/*.jsonl`).
Bound to `prefix + g` in tmux. Opens a 90% × 80% popup. Type to filter, Enter to
resume the selected session in the current pane (`cd <cwd> && claude --resume <id>`).

Listing comes from `claude-sessions-list` (Python single-process — much faster
than the per-file jq approach we started with). Worktrees show as
`repo:wt/branch` so they're scannable; full path is in the fzf preview pane.

### `claude-sessions-list`
TSV emitter consumed by `claude-tmux`. Walks `~/.claude/projects/*/*.jsonl`,
extracts `cwd` + first user message + age in a single pass per file, outputs
`age \t short_label \t session_id \t preview \t full_cwd`. Sorts newest first.

### `claude-search`
Full-text search across **all** session transcripts (1+ GB of past Claude work).
Walks `~/.claude/projects/*/*.jsonl` looking for substring matches in user/assistant
messages, outputs the same TSV shape as `claude-sessions-list` so it composes
with fzf. The `cs-find <query>` zsh function pipes its output into fzf →
selection → resume.

### `claude-tail`
Live-tail a session transcript with formatted output (`👤 user`, `🤖 assistant`,
`ⓘ system`, tool calls collapsed to `[tool: name]`). Auto-detects the session
for the current TTY, or accepts a session UUID prefix. Useful for monitoring
an agent-team teammate from another pane.

### `claude-today`
Standup-prep summary: today's sessions grouped by project (cwd), with start/end
times, durations, and first user message. Flags: `--yesterday`, `--days N`.

### `claude-worktrees`
Table of git worktrees in the current repo, with last commit age and Claude
session activity per worktree (`●` = active in last hour, `○` = active in
last 24h). Handles both `.worktrees/` (manual) and `.claude/worktrees/`
(Claude-managed agent worktrees). Aliased as `cw-list`.

### `claude-prune-worktrees`
Classify worktrees as `DIRTY`, `MERGED`, `STALE`, or `ACTIVE`. Default mode
is dry-run. `--remove merged` (or `stale`, or `merged,stale`) actually removes
after interactive confirmation. NEVER touches DIRTY worktrees. Parallelized
via `ThreadPoolExecutor` for speed (~6s on a 55-worktree repo). Aliased as
`cw-prune`.

## tmux-claude-save stack

This is the four-script chain that survives Mac restarts and reopens you on
exactly the Claude conversations you had open before.

### `tmux-claude-state`
Captures lead-claude pane → session-id mapping at tmux save time. Hooked into
`@resurrect-hook-post-save-all` in `tmux/tmux.conf`. Walks all panes via `tmux
list-panes -a`, finds ones running `claude` (via `ps -o command= -t <tty>`),
looks up the session ID via the most-recent `.jsonl` for that pane's cwd,
writes a TSV sidecar to `~/.tmux/resurrect/claude-state.txt`.

Skips agent panes (those with `--agent-id` in their command line) — they
already restore via tmux-resurrect's `@resurrect-processes` because their
full command line is captured.

### `tmux-claude-resurrect-restore`
Hooked into `@resurrect-hook-post-restore-all`. After tmux-resurrect restores
the layout (panes/windows/sessions), this reads the sidecar and for each
lead-claude entry sends `cd <cwd> && cr <session_id>` to the matching pane
via `tmux send-keys`. Pane resolution is by `(session_name, window, pane_index)`
which is stable across restarts; `pane_id` is not.

The `cr` zsh alias resolves to `claude --resume`, defined in `~/.zshrc`.

### `tmux-claude-prune`
Disk hygiene. Archive Claude session jsonls older than N days to
`~/.claude/projects-archive/` (recoverable). Default: dry-run, 30-day cutoff.
Useful because `~/.claude/projects/` accumulates indefinitely (subagent
transcripts especially). Skips files with a `.pinned` sibling (see `claude-pin`).

## Observability

### `claude-today`
Standup-prep summary: today's sessions grouped by project (cwd), with start/end
times, durations, and first user message. Flags: `--yesterday`, `--days N`.
Aliased as `cc-today` / `cc-yesterday`.

### `claude-touched`
Files Claude has edited today, ranked by edit frequency. Tool-type breakdown
shows `Edit×32, Write×1` style attribution per file.

### `claude-stats`
Token usage statistics: input / output / cache reads / cache creates per session.
Surfaces cache hit rate (a key cost-efficiency metric — typically >85% on this
setup). Top-N sessions by total tokens.

### `claude-running`
Real-time view of every Claude pane in flight. Distinguishes lead sessions (◆)
from agent teammates (◇), groups by tmux session, shows `--agent-id` labels.
Useful for "what teams are running right now?" sanity checks.

### `claude-dirty`
Quick scan of which worktrees have uncommitted changes. Lighter than full
`cw-prune` classification — uses `git diff --quiet HEAD` for speed.
Aliased as `cc-dirty`. End-of-day "what should I commit?" tool.

## Archive

### `claude-export`
Renders any session's transcript as readable markdown. Useful for sharing,
documentation, or preserving a reference conversation before pruning.
Flags: `-o file.md`, `--no-tools` (skip tool blocks).

### `claude-pin`
Protect a session from `tmux-claude-prune`. Writes `<session-id>.pinned`
sibling file. Modes: pin (default), `--unpin`, `--list`. Prefix matching
(>=4 chars).

## Statusline

### `tmux-cell-daemon`
Background process that keeps the window-status cell cache fresh. Started
automatically by tmux at server boot via `run-shell` hook. PID-file guard
prevents duplicate instances. Sleeps 4s between cache refreshes; exits
when the tmux server isn't reachable.

### `tmux-window-cell`
Fast cache reader (called per-window from `window-status-format`). Falls back
to spawning the daemon if the cache is stale. ~50ms warm-path cost (just an
awk lookup).

### `tmux-save-dot`
Single-glyph indicator for tmux-resurrect / continuum auto-save freshness:
●  bright green (<1m) · ●  green (<6m) · ◐ yellow (<15m) · ○ red (>15m).
Drop into status-right.

### `tmux-fleet-count`
Global count of active Claude agent panes across all tmux sessions
(`◇N` when N > 0). Single-pass `ps` walk, cached 4s. Goes in status-right,
NOT per-window (per-window would oscillate cell width).

### `tmux-mute-indicator`
Shows `🔇` in status-right when `~/.claude/.muted` exists (i.e. when
`cc-mute on` is active). Reminder so you don't forget to unmute later.

## Hook plumbing

### `cc-sound`
Wrapper around `afplay` that respects the mute flag (`~/.claude/.muted`).
All 5 hooks (Notification/Stop/StopFailure/SubagentStop/PreCompact) call
this instead of `afplay` directly. The `cc-mute` zsh function manages the
flag.

## Audit + ergonomic commands

### `tmux-claude-audit`
One-shot 65+ check across the entire stack: tools on PATH, helper scripts
present and executable, Claude settings (preferredNotifChannel, hooks,
power-user toggles, additionalDirectories), tmux plugins (TPM, resurrect,
continuum), tmux.conf wiring, continuum auto-save freshness, claude-state
sidecar entries, zshrc aliases/functions, fonts, ghostty config, dotfiles
git state. Aliased as `cc-audit`. `--quiet` mode for cron / CI.

The audit is the source of truth for "is everything wired?" — every script
or alias added during the iter-1→iter-27 build-out has a corresponding check.

## Helper scripts (older, pre-iter-1)

### `tmux-claude-session`
Helper for the tmux statusline + `tmux-claude-state` script. Given a TTY,
finds the active claude process on it, walks back to its cwd, derives the
project dir name (Claude's slash/dot replacement scheme), and returns the
most recently active session ID for that pane.

### `tmux-git-info`, `tmux-short-path`, `tmux-tile-session`
Per-pane tmux statusline helpers. Pre-iter-1; not part of the Claude stack.

### `claude-dev`, `setup-github-ssh`
Project-specific helpers, unrelated to this stack.

## How it composes

```
                     ┌─────────────────────┐
                     │   tmux-resurrect    │  every 5min via continuum
                     │   (saves layout)    │
                     └──────────┬──────────┘
                                │
                  @post-save    ▼
                     ┌─────────────────────┐
                     │ tmux-claude-state   │  writes per-pane session IDs to
                     │                     │  ~/.tmux/resurrect/claude-state.txt
                     └─────────────────────┘
                                
                          (Mac restart)

                                
                  @post-restore ▼
                     ┌─────────────────────────┐
                     │ tmux-claude-           │  reads sidecar, sends `cd <cwd>
                     │ resurrect-restore       │  && cr <id>` to each pane
                     └─────────────────────────┘

User-driven:
  cw <branch>     → create worktree + new claude window
  cw-list         → see all worktrees + claude activity
  cw-prune        → safely cleanup merged/stale worktrees
  cs              → fzf picker over all past sessions (claude-tmux)
  cs-find <q>     → search transcripts → fzf → resume
  cc-tail [<id>]  → live-tail an agent's progress from another pane
  cc-today        → standup-prep summary
  cc-status       → one-screen system dashboard
  cc-audit        → 68-check verifier
  cc-fix          → auto-resolve fixable warns (e.g., stale continuum save)
  cc-help         → searchable cheat sheet of all of the above
```

Configuration files this depends on:
- `~/.config/ghostty/config` — theme, font, shell-integration-features
- `~/.tmux.conf` — TPM + resurrect/continuum + post-save/post-restore hooks + keybinds
- `~/.zshrc` — starship init, aliases, functions, hooks
- `~/.claude/settings.json` — preferredNotifChannel, 4 hooks, power-user toggles
- `~/.claude/statusline-command.sh` — Claude TUI statusline (model, repo, save:Xm, ctx:%)
