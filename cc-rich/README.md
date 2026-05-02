# cc-rich

Rich tmux-popup overlay for Claude Code sessions: markdown rendering,
syntax highlighting, branch awareness, click-to-fork, click-to-merge.

## Build & install

```bash
cd ~/dev/dotfiles/cc-rich
make install install-shim
```

Installs the Go binary at `~/.local/bin/cc-rich` and a shim at `~/bin/cc-rich`.

## Usage

Three tmux key bindings (prefix is `Ctrl-a`):

| Keys | What it does |
|---|---|
| `Ctrl-a R` | Open the rich viewer for the current pane's Claude session |
| `Ctrl-a B` | Browse all known sessions |
| `Ctrl-a M` | Open the merge composer for the current pane's session |

(Note: `Ctrl-a Ctrl-r` is preserved for tmux-resurrect's restore.)

Inside the viewer:

- `j` / `k` — navigate
- `1` — fork: resume + branch from this message (new tmux window)
- `2` — fork: replay this message as a fresh prompt (new tmux window)
- `4` — quote this message into `~/.claude/buffer.md`
- `m` — open merge composer
- `q` / `Esc` — close

## Architecture

See [`docs/superpowers/specs/2026-05-01-cc-rich-design.md`](../docs/superpowers/specs/2026-05-01-cc-rich-design.md).

## Tests

```bash
make test
```
