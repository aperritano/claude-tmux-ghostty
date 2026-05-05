---
description: "tmux + Ghostty scrollback, selection, and paste reference"
---

# Scrolling, selecting, pasting

Display this reference to the user:

## TL;DR

| I want to… | Do this |
|---|---|
| Scroll back through output | Mouse wheel — auto-enters copy-mode |
| Select text without losing scroll position | Click + drag — copies to OS clipboard, exits copy-mode |
| Paste OS clipboard into prompt | `Cmd-V` (Ghostty passthrough) |
| Paste tmux buffer (last selection) | `Ctrl-a` then `]` |
| Bypass tmux entirely (full Ghostty selection) | Hold `Shift` while dragging |
| Toggle tmux mouse off | `Ctrl-a` then `m` |
| Exit copy-mode | `q` or `Esc` |

## Copy-mode (vi-style — matches editorMode=vim)

Enter with mouse-wheel-up, `Ctrl-a [`, or `PgUp`.

| Action | Key |
|---|---|
| Move cursor | `h` `j` `k` `l` |
| Word forward / back | `w` / `b` |
| Half-page up / down | `Ctrl-u` / `Ctrl-d` |
| Full-page up / down | `Ctrl-b` / `Ctrl-f` |
| Top / bottom of buffer | `g` / `G` |
| Search forward | `/pattern` then `n` / `N` |
| Search back | `?pattern` then `n` / `N` |
| Start selection | `v` (or just click+drag) |
| Yank selection | `y` (auto-pipes to `pbcopy`) |
| Quit copy-mode | `q` |

## Why selection used to jump

Before iter-51, click-and-drag in the visible region triggered tmux's "snap to prompt" — the view jerked to the bottom and your selection was wiped. Fix is at `tmux.conf:30`: a root-level `MouseDrag1Pane` binding that auto-enters copy-mode on the *first* drag pixel, so the view stays put.

## Why paste used to garble

`zsh-syntax-highlighting` re-tokenizes the entire buffer on every keypress. Pasting a 5KB blob = O(n²) work = visible corruption. Cap is `ZSH_HIGHLIGHT_MAXLENGTH=300` in `.zshrc` — paste >300 chars and highlighting just stops, paste lands clean.

## Gotchas

- **Mouse-wheel in Claude Code TUI**: the TUI consumes scroll events for its own scrollback. To scroll *tmux* history above the TUI, exit Claude (`Ctrl-d`) or split a new pane.
- **Cmd-C inside tmux**: only copies what's *visually selected by Ghostty*. If tmux is consuming the drag, use Shift+drag to bypass.
- **Bracketed paste**: tmux passes paste sequences through, so multi-line paste preserves newlines as Enter — be careful pasting commands with embedded `\n`.
