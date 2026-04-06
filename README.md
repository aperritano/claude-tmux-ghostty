# Dotfiles

macOS dev environment built around Ghostty, tmux, and Claude Code. Everything is symlinked from this repo via `install.sh`.

## What's Here

```
dotfiles/
├── ghostty/
│   ├── config                  # Alien Blood theme, NotoMono Nerd Font, tmux passthrough
│   └── themes/
│       └── Green CRT           # Custom CRT phosphor theme (alternate)
├── tmux/
│   ├── tmux.conf               # Alien Blood status bar, vim nav, Claude Code bindings
│   └── tile-toggle.conf        # Standalone tile/untile snippet
├── zsh/
│   └── zshrc                   # Oh My Zsh + Powerlevel10k, tmux auto-attach, lazy nvm/pyenv
├── vim/
│   └── vimrc                   # Gruvbox, airline, NERDTree, Copilot, ALE
├── claude/
│   ├── CLAUDE.md               # Global instructions — agent teams, DEP, evidence gathering
│   ├── settings.json           # Plugins, permissions, agent teams, opus model, plan mode
│   ├── settings.local.json     # Local permission overrides
│   ├── statusline-command.sh   # Status line — dir, git, model, context % with traffic-light
│   └── rules/
│       └── deterministic-execution-protocol.md   # 11-section verified execution protocol
├── bin/
│   ├── claude-dev              # Named tmux session launcher for Claude Code projects
│   ├── tmux-claude-session     # Session ID + active sub-agent count for status bar
│   ├── tmux-git-info           # Branch + dirty indicator (worktree-aware)
│   ├── tmux-kill-teammate-pane # Auto-close teammate panes when agents finish
│   ├── tmux-pane-label         # Pane border label — repo, branch, ahead/behind
│   ├── tmux-session-list       # Multi-session status-left renderer
│   ├── tmux-short-path         # Abbreviated path display (worktree-aware)
│   ├── tmux-tile-session       # Toggle all windows into tiled panes and back
│   └── tmux-tutorial           # Interactive walkthrough of the full setup
└── install.sh                  # Symlinks everything into place
```

## Install

```bash
git clone <this-repo> ~/dev/dotfiles
cd ~/dev/dotfiles
./install.sh
tmux source ~/.tmux.conf
```

Existing files are backed up to `*.bak` before symlinking.

### Prerequisites

| Dependency | Install |
|---|---|
| [Ghostty](https://ghostty.org) | Download from ghostty.org |
| [NotoMono Nerd Font](https://www.nerdfonts.com/) | `brew install --cask font-noto-mono-nerd-font` |
| tmux 3.3+ | `brew install tmux` |
| [Oh My Zsh](https://ohmyz.sh/) | `sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"` |
| [Powerlevel10k](https://github.com/romkatv/powerlevel10k) | `git clone --depth=1 https://github.com/romkatv/powerlevel10k.git ${ZSH_CUSTOM}/themes/powerlevel10k` |
| [zsh-autosuggestions](https://github.com/zsh-users/zsh-autosuggestions) | `git clone https://github.com/zsh-users/zsh-autosuggestions ${ZSH_CUSTOM}/plugins/zsh-autosuggestions` |
| [zsh-syntax-highlighting](https://github.com/zsh-users/zsh-syntax-highlighting) | `git clone https://github.com/zsh-users/zsh-syntax-highlighting ${ZSH_CUSTOM}/plugins/zsh-syntax-highlighting` |
| fzf | `brew install fzf` |
| pyenv | `brew install pyenv` |
| nvm | `brew install nvm` |
| [Claude Code](https://claude.ai/code) | `npm install -g @anthropic-ai/claude-code` |
| [claude-tmux](https://github.com/anthropics/claude-code/tree/main/packages/claude-tmux) | `cargo install claude-tmux` (for `Ctrl-a g` session manager) |
| [vim-plug](https://github.com/junegunn/vim-plug) | `curl -fLo ~/.vim/autoload/plug.vim --create-dirs https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim` |

After install, run `p10k configure` to set up Powerlevel10k and `vim +PlugInstall` for vim plugins.

## Shell

Oh My Zsh with Powerlevel10k lean prompt. Plugins: git, z, sudo, fzf, autosuggestions, syntax-highlighting, colored-man-pages, history-substring-search.

Key features in `.zshrc`:
- **Auto-attach tmux** when opening Ghostty (creates/reattaches `main` session)
- **`cc [name]`** — launch Claude Code with a named tmux tab
- **`claude-dev [path]`** — full dev session: Claude (65%) + shell (35%) + extra shell tab
- **Lazy-loaded** nvm and pyenv (fast shell startup)

## tmux Key Bindings

Prefix is `Ctrl-a`.

### Splits and Navigation

| Binding | Action |
|---|---|
| `Ctrl-a j` | Split right |
| `Ctrl-a i` | Split down |
| `Ctrl-h/j/k/l` | Navigate panes (no prefix needed) |
| `Ctrl-a H/J/K/L` | Resize panes |
| `Ctrl-a z` | Zoom/unzoom pane |
| `Ctrl-a =` | Equalize (tiled layout) |
| `Ctrl-a m` | Toggle mouse (tmux mouse vs native Ghostty selection) |

### Windows and Sessions

| Binding | Action |
|---|---|
| `Ctrl-a s` | New shell window |
| `Ctrl-a c` | New Claude Code window |
| `Ctrl-a C` | Claude dev layout (65/35 split) |
| `Ctrl-a n/p` | Next/previous window |
| `Ctrl-a 1-9` | Jump to window |
| `Ctrl-a R` | Rename current session |
| `Ctrl-a Tab` | Next session |
| `Ctrl-a E` | Tile/untile toggle (expose all windows as panes) |

### Claude Code

| Binding | Action |
|---|---|
| `Ctrl-a g` | Claude session manager popup |
| `Ctrl-a y` | Copy Claude session ID to clipboard |
| `Ctrl-a p` | Copy pane path to clipboard |
| `Ctrl-a X` | Kill orphaned agent panes |
| `Ctrl-a ?` | Cheat sheet |

Right-click session names or window tabs in the status bar for context menus.

## Claude Code Status Line

The status line script receives JSON from Claude Code and displays:

| Segment | Source Field | Color |
|---|---|---|
| `user@host` | system | green |
| Directory | `workspace.current_dir` | blue |
| Git branch + sync | git CLI | yellow |
| Model | `model.display_name` | cyan |
| Session name | `session_name` | magenta |
| Context usage | `context_window.used_percentage` | green/yellow/red (traffic-light) |

Additional fields available but not currently displayed: `cost.total_cost_usd`, `cost.total_lines_added`, `rate_limits.five_hour.used_percentage`, `vim.mode`, `worktree.name`.

## Claude Code Agent Teams

The `CLAUDE.md` instructions enforce tmux-based agent teams for parallel work:

1. `TeamCreate` to create a team
2. `Agent` tool with `team_name` to spawn teammates in tmux panes
3. `TaskCreate` / `SendMessage` to coordinate

Each teammate gets its own visible tmux pane. The `tmux-kill-teammate-pane` script auto-cleans panes when agents finish.

## Deterministic Execution Protocol

The DEP (`claude/rules/`) is an 11-section protocol that governs how Claude Code executes multi-step tasks:

1. Model-First Reasoning
2. Step-Level Verification
3. Constraint Re-Verification
4. Grounded Verification (never trust memory)
5. Chain-of-Verification
6. Explicit Uncertainty
7. State Tracker
8. Consensus on Judgment Calls
9. Plan Drift Detection
10. Post-Task Retrospective
11. Self-Correcting Convergence Loop

Scales by complexity: single edits use sections 4-6, multi-phase builds use all 11.
