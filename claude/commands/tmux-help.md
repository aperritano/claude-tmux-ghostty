---
description: "Show tmux pane navigation shortcuts for agent teams"
---

# tmux Pane Navigation

Display this reference to the user:

## Jumping Between Panes

| Action | Keys |
|---|---|
| Next pane (cycle forward) | `Ctrl-b` then `o` |
| Previous pane (last active) | `Ctrl-b` then `;` |
| Jump to pane by number | `Ctrl-b` then `q` (shows numbers, press number to jump) |
| Focus pane left | `Ctrl-b` then `Left` |
| Focus pane right | `Ctrl-b` then `Right` |
| Focus pane up | `Ctrl-b` then `Up` |
| Focus pane down | `Ctrl-b` then `Down` |

## Zoom and Layout

| Action | Keys |
|---|---|
| Zoom / unzoom current pane | `Ctrl-b` then `z` |
| Cycle layouts (tiled, even-h, even-v, etc.) | `Ctrl-b` then `Space` |
| Swap with next pane | `Ctrl-b` then `}` |
| Swap with previous pane | `Ctrl-b` then `{` |

## Windows (tabs)

| Action | Keys |
|---|---|
| Next window | `Ctrl-b` then `n` |
| Previous window | `Ctrl-b` then `p` |
| Jump to window by number | `Ctrl-b` then `0`-`9` |
| List all windows | `Ctrl-b` then `w` |

## Session Management

| Action | Keys |
|---|---|
| Detach from session | `Ctrl-b` then `d` |
| List sessions | `tmux ls` |
| Attach to session | `tmux attach -t <name>` |
| Kill current pane | `Ctrl-b` then `x` |

## Quick Tips

- **Lost?** `Ctrl-b` then `q` flashes pane numbers — great for finding agent teammates
- **Need focus?** `Ctrl-b` then `z` zooms one pane to fullscreen, same combo unzooms
- **Rearrange?** `Ctrl-b` then `Space` cycles through tiled/horizontal/vertical layouts
