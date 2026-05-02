# cc-rich — agent context

A tmux-popup-overlay TUI for Claude Code sessions: rich rendering, branch
awareness, click-to-fork, click-to-merge.

## Tech stack

- **Language:** Go 1.21+ (toolchain-pinned to whatever `go.mod` says)
- **Module path:** `github.com/aperritano/cc-rich`
- **TUI:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss) + [Glamour](https://github.com/charmbracelet/glamour)
- **Filesystem watch:** [fsnotify](https://github.com/fsnotify/fsnotify)
- **TUI testing:** `github.com/charmbracelet/x/exp/teatest`
- **Shims:** `cc-rich-shim`, `cc-replay-shim` (bash 3.2 portable)

## Layout

```
cc-rich/
├── cmd/cc-rich/main.go       # entry — flag parsing + Bubble Tea launch
├── internal/sessiontree/     # data layer (JSONL → tree); pure, no I/O downstream
├── internal/view/            # Bubble Tea models (Conversation, BranchList, MergeComposer)
├── internal/actions/         # side-effect dispatcher (Fork, QuoteToBuffer, WriteMergeBuffer)
├── spike/spike-f1.sh         # F1 mechanics probe (kept for re-verification)
├── e2e/e2e-test.sh           # integration smoke (17 cases, run anytime)
├── Makefile                  # build / install / install-shim / test / clean
└── README.md                 # build + usage
```

Companion docs (load these when working on cc-rich):
- Spec: `../docs/superpowers/specs/2026-05-01-cc-rich-design.md`
- Plan: `../docs/superpowers/plans/2026-05-01-cc-rich.md`

## Commands

```bash
make build              # → ./bin/cc-rich
make install            # → ~/.local/bin/cc-rich
make install-shim       # → ~/bin/cc-rich (symlink to dotfiles/bin/cc-rich-shim)
make test               # go test ./...
go vet ./...            # always run before commit
bash e2e/e2e-test.sh    # 17-case integration smoke (binary + shims + tmux binds)
```

## Hard constraints (carry these into every change)

1. **No Python in any hot path.** Python startup costs ~4 seconds on this Mac (enterprise AV scans every interpreter exec). The cc-burn-rate rewrite from Python → bash+jq+awk is the canonical example. Use Go for binaries; bash+jq+awk for shell tools.
2. **Bash 3.2 portability** for any shell script (macOS default). No `mapfile`, `declare -A`, `${var,,}`, or `${var^^}`.
3. **Sub-100ms cold start** is a hard requirement for the binary. Bubble Tea + Lipgloss + Glamour collectively achieve this; don't add Python sidecars.
4. **Atomic file writes** for any mutation: write to `path.tmp`, `os.Rename` to `path`. Patterns in `internal/actions/{fork,quote,merge}.go`.
5. **Audit-as-living-spec.** Every shipped binary needs an entry in `~/dev/dotfiles/bin/tmux-claude-audit`'s helper-script loop AND at least one regression test in `~/dev/dotfiles/bin/tmux-claude-test`.
6. **Don't touch `~/.claude/statusline-command.sh` or any path under `~/.claude/`** — managed-enterprise territory; gets overwritten on managed sync. User customizations belong in tmux territory.

## Module boundaries (enforce)

| Module | Knows about | Doesn't know about |
|---|---|---|
| `sessiontree` | JSON, file paths, parent/child tree | TUI, tmux, Glamour, Lipgloss |
| `view` | Models, Bubble Tea messages, Lipgloss styles, Glamour | tmux command shape, file I/O (gets data via `sessiontree.Tree`) |
| `actions` | `Runner` interface, file paths, command-line shape | TUI, Bubble Tea, JSON content |
| `cmd/cc-rich/main.go` | Wires the above + parses flags | Anything domain-specific (delegates) |

If a change starts crossing these lines, stop and reconsider.

## Patterns to follow

- **Side effects through `Runner`.** `actions` package never calls `os/exec` directly except in `DefaultRunner.Cmd`. Tests pass a mock — never bypass.
- **TDD per task.** Write the failing test, run to confirm fail, implement, run to confirm pass, commit. Every task in the plan follows this; new work should too.
- **One commit per task** with conventional-commit prefixes: `feat(cc-rich/<pkg>):`, `fix(cc-rich):`, `test(cc-rich/<pkg>):`, `docs(cc-rich):`, `refactor(cc-rich/<pkg>):`.
- **`go vet` before commit.** Catches the things `go build` doesn't.

## Gotchas (learned the hard way; do not relearn)

- **`claude --resume <synthesized-sid>` returns "No conversation found".** Claude Code's session index is internal — dropping a JSONL into `~/.claude/projects/<dir>/` does NOT register it. Spike: `spike/spike-f1.sh` (commit `d1ad3ec`). F1 forks therefore use a synthesized preamble + `cc-replay-shim`, not `claude --resume`.
- **`tmux-claude-session` has two output modes.** Default = statusline-friendly (`│ claude:abc12345`, truncated to 8 chars). `--bare` = full UUID, no decoration. Programmatic consumers (cc-rich) MUST use `--bare`. Bug history: commit `8139859`.
- **Bubble Tea needs a real TTY.** Popup invocations get one. Direct shell invocations (Bash tool, `cc-rich --pane %X | head`) don't — the binary will print `could not open a new TTY: open /dev/tty: device not configured`. Test the popup with `tmux display-popup -E "$HOME/bin/cc-rich --pane #{pane_id} 2>/tmp/log; sleep 5"` to capture stderr.
- **Tmux popup binds must use `$HOME/bin/cc-rich` (full path), not bare `cc-rich`.** `display-popup` spawns a non-interactive shell that does NOT source `~/.zshrc`, so `~/bin` isn't on PATH. Bare command → exit 127 (`command not found`) → popup flashes and disappears. Same trap applies to `run-shell`. The audit's "popup binds use full paths" check guards against this regressing.
- **cc-rich is now a sidebar, not a popup.** Binds R/B/M call `bin/cc-rich-sidebar` which `split-window -h -l 35%`s a side-pane and tags its title `@cc-rich-sidebar`. Second invocation of the same key finds the tagged pane and `kill-pane`s it (toggle). **Don't bind `run-shell "$HOME/bin/cc-rich --pane ..."` directly** — `run-shell` runs detached with no TTY, and Bubble Tea will exit with `could not open a new TTY`. The split-window dance gives the new pane a real TTY. Audit guard: each of R/B/M must reference `cc-rich-sidebar`.
- **Tmux key bindings:** `Ctrl-a R` (pane), `Ctrl-a B` (browse), `Ctrl-a M` (merge). **Not `Ctrl-a Ctrl-r`** — that belongs to tmux-resurrect's restore (load-bearing in the save/restore stack). Conflict-fix commit `29d4fe0`.
- **Per-pane caching.** The `tmux-pane-header` script caches at `/tmp/tmux-pane-header-$UID/` (per-pane keyed by id+cwd). 4s TTL. Never bypass — git+ps probes per pane per refresh add up fast.
- **lipgloss is a transitive pre-release** (pulled in by glamour). `go.mod` will show `v1.1.1-0.<date>-...`. Don't pin to a stable lipgloss release; it'll fight glamour.

## When you'd surface a question to the human

- A change would cross module boundaries (data ↔ view ↔ actions)
- A change requires touching `~/.claude/*` or rewriting tmux-resurrect
- A new dependency adds >5 MB to the binary or pulls in cgo
- The Bubble Tea API has changed (versions drift; check before assuming)
- A test fails for a reason you don't immediately understand — run the systematic-debugging skill, don't guess

## Verification before claiming "done"

```bash
go vet ./... && \
go test ./... && \
bash e2e/e2e-test.sh && \
make build && make install
```

All four must pass. Then test the popup interactively — `Ctrl-a R` from a pane running Claude — because no test rig drives the TUI keystrokes end-to-end.
