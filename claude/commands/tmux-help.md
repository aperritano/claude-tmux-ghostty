---
description: "Show tmux pane navigation shortcuts for agent teams"
---

# tmux Pane Navigation

Display this reference to the user:

## Jumping Between Panes

| Action | Keys |
|---|---|
| Next pane (cycle forward) | `Ctrl-a` then `o` |
| Previous pane (last active) | `Ctrl-a` then `;` |
| Jump to pane by number | `Ctrl-a` then `q` (shows numbers, press number to jump) |
| Focus pane left | `Ctrl-a` then `Left` |
| Focus pane right | `Ctrl-a` then `Right` |
| Focus pane up | `Ctrl-a` then `Up` |
| Focus pane down | `Ctrl-a` then `Down` |

## Zoom and Layout

| Action | Keys |
|---|---|
| Zoom / unzoom current pane | `Ctrl-a` then `z` |
| Cycle layouts (tiled, even-h, even-v, etc.) | `Ctrl-a` then `Space` |
| Swap with next pane | `Ctrl-a` then `}` |
| Swap with previous pane | `Ctrl-a` then `{` |

## Windows (tabs)

| Action | Keys |
|---|---|
| Next window | `Ctrl-a` then `n` |
| Previous window | `Ctrl-a` then `p` |
| Jump to window by number | `Ctrl-a` then `0`-`9` |
| List all windows | `Ctrl-a` then `w` |

## Session Management

| Action | Keys |
|---|---|
| Detach from session | `Ctrl-a` then `d` |
| List sessions | `tmux ls` |
| Attach to session | `tmux attach -t <name>` |
| Kill current pane | `Ctrl-a` then `x` |

## Quick Tips

- **Lost?** `Ctrl-a` then `q` flashes pane numbers — great for finding agent teammates
- **Need focus?** `Ctrl-a` then `z` zooms one pane to fullscreen, same combo unzooms
- **Rearrange?** `Ctrl-a` then `Space` cycles through tiled/horizontal/vertical layouts
