# bin/ — Utility Scripts

All scripts are symlinked to `~/bin/` by `install.sh`.
Scripts marked **dormant** are present in bin/ but not called from `tmux.conf`.

---

## Status-bar scripts

Called by `tmux.conf` `status-right` via `#(...)` every `status-interval` seconds.

| Script | Shell | Purpose |
|---|---|---|
| `tmux-short-path` | zsh | Abbreviated path for status bar. `wt:<name>` for worktree roots; `~/…/parent/dir` for deep paths; hard cap at 30 chars. |
| `tmux-git-info` | zsh | Branch name + dirty indicator (`*`). Worktree-aware: omits redundant prefix for linked worktrees. |
| `tmux-claude-session` | zsh | Session ID + active sub-agent count. Reads `.claude/projects/` JSONL files. **macOS-only** (`lsof`, `stat -f %m`, BSD `ps`). |

Usage in `tmux.conf`:
```
status-right "... #(~/bin/tmux-short-path '#{pane_current_path}') ... #(~/bin/tmux-git-info '#{pane_current_path}') #(~/bin/tmux-claude-session '#{pane_tty}')"
```

---

## Keybind-triggered scripts

| Script | Keybind | Shell | Purpose |
|---|---|---|---|
| `tmux-tile-session` | `Ctrl-a E` | bash | Toggle all windows into tiled panes and back. Stores original window names as pane options (`@orig_name`, `@orig_pos`) for restore. |
| `tmux-kill-teammate-pane` | via `Ctrl-a X` (`kill-pane -a`) | zsh | Kill the current pane if it is not the last pane in the window. Safety guard: exits 0 outside tmux. |

---

## Session / setup launchers

| Script | Usage | Shell | Purpose |
|---|---|---|---|
| `claude-dev` | `claude-dev [path]` | zsh | Attach or create a named tmux session for a project: Claude left (65%) + shell right (35%) + extra shell tab. |
| `setup-github-ssh` | `setup-github-ssh` | bash | Generate an ed25519 SSH key and upload it to GitHub via `gh`. Idempotent: prompts before overwriting. **macOS-only** (`ssh-add --apple-use-keychain`). |

---

## Tutorial

| Script | Usage | Purpose |
|---|---|---|
| `tmux-tutorial` | `tmux-tutorial` | Interactive step-by-step walkthrough of the full tmux + Claude Code setup. 12 steps. Run standalone in any terminal. |

---

## Dormant (not wired in `tmux.conf`)

These scripts exist and work but are not called by `tmux.conf` or any hook.

| Script | Shell | Why dormant | Action needed |
|---|---|---|---|
| `tmux-pane-label` | bash | `pane-border-status` is not set; `pane-border-format` is not configured. Output: `repo  branch ↑N ↓M`. PR #4 wires it. | Wire or leave opt-in. |
| `tmux-session-list` | bash | `status-left` uses the built-in `#{S:...}` inline format instead. Script uses hardcoded hex colors inconsistent with the ANSI palette approach in `tmux.conf`. | Retire or rewrite with ANSI colors and rewire. |

---

## Open questions (carry-forward)

- **`tmux-kill-teammate-pane` hook wiring**: README claims auto-clean on `SubagentStop`, but no hook is wired in `claude/settings.json`. Decision: wire hook or retract the README claim.
- **`tmux-session-list`**: Dead code since `#{S:...}` replaced it. Decision: remove or rewire with ANSI palette colors.
- **`tmux-pane-label`**: Built, documented, unused. Decision: enable pane-border-status or keep as opt-in.
