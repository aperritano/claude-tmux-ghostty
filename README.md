# Claude + tmux + Ghostty

macOS dev environment built around Ghostty, tmux, and Claude Code. One script bootstraps a fresh Mac from zero.

## Quick Start (New Mac)

```bash
# 1. Install Xcode command line tools (required for git)
xcode-select --install

# 2. Clone via HTTPS (SSH isn't set up yet) and bootstrap
git clone https://github.com/aperritano/claude-tmux-ghostty.git ~/dev/dotfiles
cd ~/dev/dotfiles
./install.sh

# 3. Set up SSH key and register it with GitHub
setup-github-ssh

# 4. Switch this repo to SSH
git remote set-url origin git@github.com:aperritano/claude-tmux-ghostty.git
```

The bootstrap script will:
1. Install Homebrew (if missing)
2. Install all packages via `Brewfile` (tmux, fzf, jq, pyenv, nvm, Go, Rust, Ghostty, fonts...)
3. Install Oh My Zsh + Powerlevel10k + zsh plugins
4. Symlink all configs into place (existing files backed up to `*.bak`)
5. Install vim-plug + vim plugins
6. Install Node.js LTS via nvm
7. Install Claude Code CLI and claude-tmux
8. Create `~/.env` from template

After install, complete these manual steps:
```bash
vim ~/.env               # Add your API keys
./macos/defaults.sh      # (Optional) Set macOS system preferences
```

## What's Here

```
dotfiles/
├── Brewfile                    # All Homebrew dependencies
├── install.sh                  # Full bootstrap script
├── .env.template               # API key template -> ~/.env
├── ghostty/
│   ├── config                  # Alien Blood theme, NotoMono Nerd Font, tmux passthrough
│   └── themes/
│       └── Green CRT           # Custom CRT phosphor theme (alternate)
├── tmux/
│   ├── tmux.conf               # Alien Blood status bar, vim nav, Claude Code bindings
│   └── tile-toggle.conf        # Standalone tile/untile snippet
├── zsh/
│   ├── zshrc                   # Oh My Zsh + Powerlevel10k, tmux auto-attach, lazy nvm/pyenv
│   └── p10k.zsh                # Powerlevel10k prompt configuration
├── vim/
│   └── vimrc                   # Gruvbox, airline, NERDTree, Copilot, ALE
├── git/
│   ├── gitconfig               # User, aliases, delta, rebase, rerere
│   └── gitignore_global        # Global gitignore (.DS_Store, .env, node_modules, etc.)
├── claude/
│   ├── CLAUDE.md               # Global instructions — agent teams, DEP, evidence gathering
│   ├── settings.json           # Plugins, permissions, agent teams, opus model
│   ├── settings.local.json     # Local permission overrides
│   ├── statusline-command.sh   # Status line — dir, git, model, context % with traffic-light
│   └── rules/
│       └── deterministic-execution-protocol.md   # 11-section verified execution protocol
├── macos/
│   └── defaults.sh             # System preferences (keyboard, Dock, Finder, screenshots)
├── bin/
│   ├── setup-github-ssh         # Generate SSH key and upload to GitHub
│   ├── claude-dev              # Named tmux session launcher for Claude Code projects
│   ├── tmux-claude-audit       # Living spec: audit checks for scripts, conf, README, settings
│   ├── tmux-claude-session     # Session ID + active sub-agent count for status bar
│   ├── tmux-claude-test        # Regression tests for bin/ scripts (CI-safe)
│   ├── tmux-git-info           # Branch + dirty indicator (worktree-aware)
│   ├── tmux-kill-teammate-pane # Auto-close teammate panes when agents finish
│   ├── tmux-pane-label         # Pane border label — repo, branch, ahead/behind
│   ├── tmux-session-list       # Multi-session status-left renderer
│   ├── tmux-short-path         # Abbreviated path display (worktree-aware)
│   ├── tmux-tile-session       # Toggle all windows into tiled panes and back
│   └── tmux-tutorial           # Interactive walkthrough of the full setup
└── README.md
```

## Shell

Oh My Zsh with Powerlevel10k lean prompt. Plugins: git, z, sudo, fzf, autosuggestions, syntax-highlighting, colored-man-pages, history-substring-search.

Key features in `.zshrc`:
- **Auto-attach tmux** when opening Ghostty (creates/reattaches `main` session)
- **`cc [name]`** -- launch Claude Code with a named tmux tab
- **`claude-dev [path]`** -- full dev session: Claude (65%) + shell (35%) + extra shell tab
- **Lazy-loaded** nvm and pyenv (fast shell startup)

## Git

The gitconfig includes:
- **git-delta** for side-by-side diffs with line numbers
- **rebase on pull** with auto-stash
- **rerere** enabled (remembers conflict resolutions)
- **Histogram diff** algorithm + zdiff3 conflict style
- Common aliases: `lg` (graph log), `co`, `br`, `ci`, `st`, `amend`, `unstage`

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

## macOS Defaults

Run `./macos/defaults.sh` to set:
- Fast keyboard repeat, no auto-correct/smart quotes
- Auto-hide Dock, no recents, small icons
- Finder: show hidden files, extensions, path bar, list view
- Screenshots to `~/Screenshots` as PNG
- Tap to click on trackpad

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
