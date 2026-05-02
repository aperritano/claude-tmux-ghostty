---
description: "Show tmux key bindings for this tmux + Claude Code setup"
---

# tmux Key Bindings Reference

**Prefix key: `Ctrl-a`** (this setup replaces the tmux default `Ctrl-b`)

Display this reference to the user:

## Splits

| Action | Keys |
|---|---|
| Split pane right | `Ctrl-a j` |
| Split pane below | `Ctrl-a i` |

## Pane Navigation (no prefix needed)

| Action | Keys |
|---|---|
| Move focus left | `Ctrl-h` |
| Move focus down | `Ctrl-j` |
| Move focus up | `Ctrl-k` |
| Move focus right | `Ctrl-l` |
| Zoom / unzoom current pane | `Ctrl-a z` |
| Equalize panes (tiled layout) | `Ctrl-a =` |
| Cycle layouts | `Ctrl-a Space` |
| Swap with next pane | `Ctrl-a }` |
| Swap with previous pane | `Ctrl-a {` |
| Jump to pane by number | `Ctrl-a q` (shows numbers, press number to jump) |

## Pane Resize (repeatable with prefix held)

| Action | Keys |
|---|---|
| Resize left | `Ctrl-a H` |
| Resize down | `Ctrl-a J` |
| Resize up | `Ctrl-a K` |
| Resize right | `Ctrl-a L` |

## Windows

| Action | Keys |
|---|---|
| New Claude Code pane (tiled) | `Ctrl-a c` |
| New shell pane (tiled) | `Ctrl-a s` |
| Claude dev layout (65% claude + 35% shell) | `Ctrl-a C` |
| Next window | `Ctrl-a n` |
| Jump to window by number | `Ctrl-a 1`–`9` |
| List all windows | `Ctrl-a w` |
| Kill current pane | `Ctrl-a x` |
| Kill current window | `Ctrl-a &` |

## Sessions

| Action | Keys |
|---|---|
| Next session | `Ctrl-a Tab` |
| Previous session | `Ctrl-a Shift-Tab` |
| Rename session | `Ctrl-a R` |
| Detach from session | `Ctrl-a d` |

## Claude Code

| Action | Keys |
|---|---|
| Claude session manager popup | `Ctrl-a g` |
| Copy Claude session ID to clipboard | `Ctrl-a y` |
| Copy current pane path to clipboard | `Ctrl-a p` |
| Tile / untile all windows as panes | `Ctrl-a E` |
| Kill all other panes (clean up agent panes) | `Ctrl-a X` |
| Toggle mouse mode | `Ctrl-a m` |

## Config

| Action | Keys |
|---|---|
| Reload tmux config | `Ctrl-a r` |
| Show cheat sheet popup | `Ctrl-a ?` |

## Quick Tips

- **Lost in a tiled layout?** `Ctrl-a q` flashes pane numbers — press the digit to jump to that pane
- **Need focus?** `Ctrl-a z` zooms one pane fullscreen; same combo unzooms
- **Start a Claude dev session?** `Ctrl-a C` opens a new window: Claude Code left (65%) + shell right (35%)
- **Clean up finished agent panes?** `Ctrl-a X` kills every pane except the current one
- **Select text in Ghostty?** `Ctrl-a m` toggles mouse — off lets Ghostty handle native selection, on enables tmux scroll and pane resize by mouse
