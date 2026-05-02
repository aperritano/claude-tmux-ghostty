---
description: "Claude Code + tmux + Ghostty workflow cheat sheet (custom shortcuts shipped via dotfiles)"
---

# Claude Code Workflow Shortcuts

Display this reference to the user. These are Anthony's custom shell helpers
shipped via `~/dev/dotfiles/`. Run `cc-doctor` to verify all are wired.

## Pickers / search

| Command | What it does |
|---|---|
| `cs` | fzf picker over all past Claude sessions |
| `cs-recent [hours]` | picker filtered to last N hours (default 4) |
| `cs-here [prefix]` | picker filtered to current project (sessions under `$PWD`) |
| `cs-find <query>` | full-text search across all transcripts → fzf → resume |
| `cc-tail [<id>]` | live-tail another pane's session in formatted output |

## Worktree workflow

| Command | What it does |
|---|---|
| `cw <branch>` | create-or-attach worktree, open Claude in new tmux window |
| `cw-go` | fzf-jump into existing worktree (switches if window already open) |
| `cw-list` | table of worktrees with branch + age + Claude activity (●/○) |
| `cw-prune` | classify DIRTY / MERGED / STALE / ACTIVE; safe-remove |
| `cc-dirty` | quick scan of worktrees with uncommitted changes |

## Observability

| Command | What it does |
|---|---|
| `cc-status` | dashboard: hooks + save-stack + worktrees + audit |
| `cc-today` / `cc-yesterday` | sessions today/yesterday grouped by project |
| `cc-touched` | files Claude edited today, ranked by edit frequency |
| `cc-stats` | token usage today: input/output/cache hit rate |
| `cc-running` | every active Claude pane (lead ◆ vs agent ◇) |
| `cc-jump [substr]` | fzf-jump to any active Claude pane (auto-selects on unique match) |
| `cc-rich` | open the rich session viewer for the current pane (also `Ctrl-a R`) |
| `cc-rich --browse` | session picker over all transcripts |

## Health / curation

| Command | What it does |
|---|---|
| `cc-audit` | 90+ check system verifier (config wiring) |
| `cc-test` | regression suite for helper scripts (20+ tests) |
| `cc-doctor` | composite: audit + test + auto-fix |
| `cc-fix` | auto-resolve fixable warns (e.g. stale continuum save) |
| `cc-pin <id>` | protect a session from prune (--list, --unpin) |
| `cc-prune` | archive sessions older than N days (dry-run default) |
| `cc-mute [on/off]` | silence hook sounds (banner notifications still fire) |
| `cc-test-sounds` | play all 5 hook sounds in sequence |

## Convenience

| Command | What it does |
|---|---|
| `cc-help` | this cheat sheet (fzf-searchable in shell) |
| `/scrolling` | tmux + Ghostty scrollback / selection / paste reference |
| `/tmux-help` | tmux pane / window / session navigation shortcuts |
| `cc-settings` | edit `~/.claude/settings.json` |
| `cc-memory` | edit auto-memory `MEMORY.md` |
| `cc-projects` | list most-recent project directories |
| `cc-export <id> [-o file.md]` | render a session as readable markdown |
| `ch` | `claude --model haiku-4-5` (fast/cheap one-offs) |
| `cr` | `claude --resume` (built-in picker) |

## tmux keybinds (prefix = Ctrl-a)

| Keys | What it does |
|---|---|
| `Ctrl-a g` | Claude session picker popup (fzf, resume saved) |
| `Ctrl-a a` | cc-jump: fzf-jump to a RUNNING Claude pane |
| `Ctrl-a h` | system status dashboard popup |
| `Ctrl-a c` | new Claude pane in current window |
| `Ctrl-a C` | new window with Claude+shell layout |
| `Ctrl-a n` / `N` | new tab / new session (prompts for name) |
| `Ctrl-a Ctrl-s` | save tmux state (resurrect) |
| `Ctrl-a Ctrl-r` | restore tmux state (resurrect) |
| `Ctrl-a R` | cc-rich: rich popup view of the active session (with [1][2][4][m] fork buttons) |
| `Ctrl-a B` | cc-rich: browse all known sessions |
| `Ctrl-a M` | cc-rich: open merge composer for the active session |
| `Ctrl-a ?` | full tmux cheat sheet |

## Audio cues (default; mute via `cc-mute`)

| Event | Sound |
|---|---|
| Notification (Claude needs attention) | Glass.aiff + macOS banner |
| Stop (turn complete) | Pop.aiff |
| StopFailure (turn errored) | Funk.aiff + banner |
| SubagentStop (teammate finished) | Submarine.aiff @30% |
| PreCompact (context compression imminent) | Tink.aiff |

## What runs automatically (no command needed)

- continuum auto-saves tmux state every 5 min
- post-restore hook resumes Claude conversations after Mac restart
- statusline shows: model · branch · save freshness (●/◐/○) · ctx % · agent count (◇N) · 🔇 if muted
- per-window ⚠ marker fires on Notification hook (the pane needing input), clears on Stop hook
- tmux-cell-daemon refreshes window-status cell cache in background
