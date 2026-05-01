# cc-rich: Rich tmux-popup overlay for Claude Code sessions

**Date:** 2026-05-01
**Status:** Approved design — pending implementation plan
**Author:** Anthony Perritano (with Claude Code, brainstorming session)

## Problem

Claude Code's terminal UI shows the active conversation as a flowing stream of turns. Two everyday workflows are missing:

1. **Rich, scannable rendering of past turns** — markdown styling, syntax-highlighted code blocks, message metadata (timestamps, models, token usage), branch awareness. Today the only options are `cc-tail` (live formatted text), `cc-export` (markdown to file), or scrolling through the live TUI. None give a "claude desktop"-like reading experience.

2. **"Branch a thread" without leaving the terminal** — when you want to try a different direction from message N of a conversation, today you have to manually copy context, run `claude` afresh, and paste. There's no in-place fork affordance and no path to merge discoveries from a sibling branch back into the main thread.

This spec describes `cc-rich`: a tmux-popup-overlay TUI that renders sessions richly and provides keyboard-driven fork + merge actions, all without leaving the user's existing tmux/Claude workflow.

## Goals

- Stay inside tmux: invoked via `display-popup`, no Cmd-Tab, no second app, no web browser.
- Match "claude desktop" rendering quality within terminal limits: markdown via Glamour, syntax highlighting, message metadata, branch tree.
- Click-to-fork (or keyboard-driven fork) that creates real Claude sessions, not just hyperlinks.
- Click-to-merge that brings selected messages from a sibling branch into the active session as quoted context for the next prompt.
- Sub-100ms cold start. No Python (the host enterprise's AV adds a 4s penalty to every Python invocation).
- Clean module boundaries so the data layer, TUI, and side effects are independently testable.

## Non-goals (v1)

- LLM-summarized merges (Haiku call to summarize a branch before merging) — future.
- Cross-machine session lineage — future, requires `cc-brain-sync` extension.
- Editing past messages — append-only by design.
- Full visual graph for >2 levels of branching — flat list for v1.
- Replacement for `cc-tail` / `cc-export` — those keep their existing roles.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Claude pane (Window 5)        │  Claude pane (Window 7)    │
│  session abc...                │  forked session def...     │
└─────────────────────────────────────────────────────────────┘
                  ↑ Ctrl-a R                  ↑ Ctrl-a R
                  │                           │
                  ↓                           ↓
        ┌──────────────────────────────────────────┐
        │   tmux display-popup overlay (any pane)  │
        │   ┌────────────────────────────────────┐ │
        │   │  cc-rich (Go + Bubble Tea TUI)     │ │
        │   │  ┌─────────────┬────────────────┐  │ │
        │   │  │  branches   │  rendered      │  │ │
        │   │  │  ▸ abc      │  conversation  │  │ │
        │   │  │    └─ def   │  with F1/F2/F4 │  │ │
        │   │  │  ▸ ghi      │  buttons +     │  │ │
        │   │  │             │  Merge action  │  │ │
        │   │  └─────────────┴────────────────┘  │ │
        │   └────────────────────────────────────┘ │
        └──────────────────────────────────────────┘
                  │ button click
                  ↓
   ┌─────────────────────────────────────────────────┐
   │  Action dispatcher (in-process)                 │
   │  ─ F1: tmux new-window "claude --resume <id>"   │
   │  ─ F2: tmux new-window "claude" + paste text    │
   │  ─ F4: append message to ~/.claude/buffer.md    │
   │  ─ Merge: write to <target>/.cc-pending-prompt  │
   └─────────────────────────────────────────────────┘
                  ↑
                  │ fsnotify
   ┌─────────────────────────────────────────────────┐
   │  ~/.claude/projects/*/*.jsonl  (transcripts)    │
   │  parent/child tree via .parentUuid edges        │
   └─────────────────────────────────────────────────┘
```

### Why a tmux popup, not a web browser

Three options were considered: text-browser-in-pane (lynx/elinks), browser-alongside-tmux (local web server + Safari), and tmux-popup-with-rich-TUI. The popup wins on:

- Stays in the tmux workflow (the user's literal request: "rich overlay in window")
- Real fork buttons (not faked HTTP-fired hyperlinks)
- Mouse OR keyboard input (tmux popups inherit mouse mode)
- Live tail without WebSocket plumbing (just fsnotify on the JSONL)
- Pane-border + statusline still visible underneath

The cost is rendering ceiling: terminal capabilities, not real CSS. With Glamour + 24-bit color + sixel/kitty graphics for diagrams, the gap to claude desktop is acceptable for a power-user workflow.

### Why Go + Bubble Tea, not Python

Python startup on the host machine costs ~4 seconds (enterprise AV scans every interpreter exec). The popup is a hot path — invoked many times per day. Go binaries start in <100ms and are immune to the AV penalty. Bubble Tea + Glamour + Lipgloss form the standard Charmbracelet stack for rich terminal UIs.

## Components

### 1. `cmd/cc-rich/main.go` — entry point

Parses flags:
- `--pane <id>` — resolve session from this tmux pane (default invocation from `bind R`)
- `--browse` — open a session picker over all running Claude sessions
- `--merge-into <pane>` — open directly to the merge composer for the pane's session
- `--headless --emit-html` (test-only) — non-TUI golden-output mode

Boots Bubble Tea with the right initial model. Exits on Esc/q (popup closes). No long-running daemon.

### 2. `internal/sessiontree` — data layer

Pure data; zero TUI deps; trivially unit-testable.

```go
type Message struct {
    UUID, ParentUUID string
    Role             string  // "user" | "assistant"
    Content          []Block // text, tool_use, tool_result, thinking
    Timestamp        time.Time
    Model            string  // claude-opus-4-7 etc.
    Usage            Usage
}

type Tree struct {
    BySID    map[string]*Message
    Children map[string][]string  // parent UUID → child UUIDs
}

func Load(path string) (*Tree, error)
func (t *Tree) Lineage(sid string) []*Message
func (t *Tree) Siblings(sid string) [][]*Message  // sibling branches across all known sessions
func (t *Tree) BranchPoints() []string  // parents with len(children) > 1
```

### 3. `internal/view` — Bubble Tea models

- `BranchListModel` (left pane): tree view of sessions sharing lineage. Cursor up/down navigates; Enter focuses a branch in the center pane.
- `ConversationModel` (center): scrollable list of rendered turns. Each turn block has `[1] [2] [4] [m]` keyboard buttons (numeric hotkeys). Glamour renders markdown.
- `MergeComposerModel` (right, opens on demand): list-checkbox of messages from a chosen sibling; `<space>` to select, `<Enter>` to commit.
- Layout via Lipgloss flexbox; resizes with the popup.
- Theme matches the user's Apple Classic palette (Lipgloss `styles.go`).

### 4. `internal/actions` — side-effect layer

```go
type Runner interface {
    Cmd(name string, args ...string) error
}

func Fork(r Runner, mode ForkMode, sourceSID, fromMsgUUID string, opts ForkOpts) error
func QuoteToBuffer(msg *Message) error
func WriteMergeBuffer(targetCWD string, citations []*Message) error
```

`Runner` is the seam for unit tests (mock impl). The default impl shells out to `tmux` and writes files atomically.

### 5. tmux glue (3 lines in `tmux.conf`)

```
bind R display-popup -E -w 90% -h 90% "cc-rich --pane #{pane_id}"
bind C-r display-popup -E -w 90% -h 90% "cc-rich --browse"
bind M display-popup -E -w 90% -h 90% "cc-rich --merge-into #{pane_id}"
```

## Fork semantics (v1 buttons per turn)

| Button | Mechanic | Use case |
|---|---|---|
| `[1]` Resume + branch (F1) | Copy `<orig-sid>.jsonl[..msgUUID]` → `<new-sid>.jsonl`; `tmux new-window "claude --resume <new-sid>"` | "Try a different direction from this point with full prior context" |
| `[2]` Replay as prompt (F2) | New window, fresh `claude`, message text injected as the first user prompt | "Start from new — explore this idea without prior context" |
| `[4]` Quote to buffer (F4) | Append message text + citation header to `~/.claude/buffer.md`. Popup stays open. | "Save this thought, no spawning yet" |
| Worktree fork (F3) | Opt-in checkbox in F1 dialog: F1 + `git worktree add` first | "Try a fix without polluting my main worktree" |

### Open implementation question (verified during iter-1)

`claude --resume <sid>` with a synthesized session id (prefix-copy of an existing JSONL) may or may not be accepted by Claude Code. If it rejects unknown session ids, F1 falls back to F2 + a synthetic preamble:

```
// continuing from msg <uuid>; prior context summary:
<last-3-msgs-summarized>
```

The user-visible behavior of "click [1] → new window with that history" is preserved either way.

## Merge feature (v1 quote-and-cite)

After F1 forks, the original and the fork are sibling descendants of the same parent. To bring discoveries back:

1. Open `cc-rich --pane <p>` (popup) on the destination pane (the one you want the merge result in).
2. Left pane shows the branch tree; sibling branches are visible.
3. Pick a sibling branch → press `m` → MergeComposer opens.
4. Check N messages from that branch with `<space>`; press `<Enter>` to commit.
5. Selected messages are formatted as a blockquote with a citation header and written to `<target-cwd>/.cc-pending-prompt`:

```markdown
// imported from branch <sid>:msg<uuid> at 2026-05-01T12:00:00Z

> [block 1: text]

> [block 2: text]
```

6. Status bar flashes: `merged 2 msgs → paste with Ctrl-a P or read from <path>`.
7. The destination Claude session is NOT auto-prompted. The user pastes from the buffer when ready (preserves the user's control over what's actually sent).

Future v2: a `Merge & Send` button that auto-injects the buffer as the next prompt; a `Merge with summary` button that calls Haiku first.

## Data flow

### Read path (popup open)

```
1. User hits Ctrl-a R in pane %5
2. tmux runs: cc-rich --pane %5
3. cc-rich queries: tmux-claude-session %tty → "claude:abc123..."
4. Resolves transcript path: ~/.claude/projects/-cwd-slug/abc123.jsonl
5. sessiontree.Load(path) → Tree
6. sessiontree.Lineage(abc123) → walks backward, finds branch ancestors, returns siblings
7. Bubble Tea renders BranchList (left) + Conversation (center)
8. fsnotify.Watcher on the JSONL → re-render on append (100ms debounce)
```

### Write paths

**F1 click:**
1. User presses `1` on a message
2. `actions.Fork(Resume, sid, msgUUID, {})`
3. Atomically copy `sid.jsonl[..msgUUID]` → `<new-sid>.jsonl`
4. Shell: `tmux new-window -t #{session_name} -c <orig-cwd> "claude --resume <new-sid>"`
5. Popup closes; user lands in new window with new fork

**F2 click:**
1. User presses `2`
2. `actions.Fork(Replay, _, msgUUID, {})`
3. Extract message text → write to `/tmp/cc-replay-<rand>.txt`
4. Shell: `tmux new-window ... "cc-replay-shim /tmp/cc-replay-<rand>.txt"`
5. Shim reads file, starts `claude` with content as initial prompt

**F4 click:**
1. User presses `4`
2. `actions.QuoteToBuffer(msg)`
3. Append to `~/.claude/buffer.md` with header + body + `---`
4. Status flash: "quoted to ~/.claude/buffer.md"
5. Popup stays open

**Merge click:**
1. User picks sibling branch B in left pane, presses `m`
2. MergeComposer opens, user checks N messages, presses Enter
3. `actions.WriteMergeBuffer(targetCWD, msgs)`
4. Format msgs as blockquote with citations
5. Atomic write to `<targetCWD>/.cc-pending-prompt`
6. Status flash: "merged N msgs → ..."
7. No auto-send

## Error handling

| Condition | Behavior |
|---|---|
| Active pane has no Claude session | Popup shows: "no Claude session — try `Ctrl-a Ctrl-r` to browse" |
| JSONL missing or unreadable | Popup shows error + path; suggest `cc-doctor` |
| Last JSONL line partial (race with live tail) | `sessiontree.Load` skips unparseable lines; logs to stderr; UI keeps last good state |
| Glamour can't render a content block | Fall back to plain `<pre>` rendering for that block; rest renders normally |
| `tmux new-window` fails | Action returns error; popup flashes red status with stderr; popup STAYS OPEN |
| Sibling lineage crosses projects | Tree shows the cross-project siblings with project tag in the label |
| Popup closed mid-fork | Action runs in goroutine with `defer` cleanup; partial JSONL writes are atomic; no half-state |

## Defensive behaviors

- **Atomic writes** for any mutation: prefix-copied JSONL, merge buffer, F4 quote buffer use `write tmp → fsync → rename`.
- **fsnotify debounce 100ms** on the watcher — bursts of appends during a tool-use sequence don't trigger 50 re-renders.
- **File-size cap** on Glamour render: messages over 100KB get truncated with a `[… truncated, see <path>]` link.
- **Read-only by default** on the source transcript: `cc-rich` never mutates `<orig-sid>.jsonl`. Forks always go to a new file.

## Edge cases

| Case | v1 decision |
|---|---|
| Session has 50k+ messages | Lazy-render: last 200 turns by default; `g`/`G` jump to top/bottom; `/<query>` search; older turns hydrate on demand |
| Branch point with 5+ children | List view in left pane (no graph); sort by mtime desc |
| Glamour renders mermaid as code (it can't draw) | v1 ships as code; v2 could spawn `mmdc` to render PNG via kitty graphics |
| Two `cc-rich` popups open simultaneously | Independent processes; each has its own watcher; no shared state |
| F1 spike fails | Fallback to F2 + synthetic preamble (see Open implementation question above) |
| Enterprise sandbox blocks `tmux new-window` from popup | Surfaces as the "tmux failed" error. User adds an allow rule. |

## Repo layout

```
~/dev/dotfiles/
├── cc-rich/                     # Go module — rich-view binary
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/cc-rich/main.go
│   ├── internal/
│   │   ├── sessiontree/{tree.go,tree_test.go,testdata/...}
│   │   ├── view/{branchlist.go,conversation.go,merge.go,styles.go,view_test.go}
│   │   └── actions/{runner.go,fork.go,quote.go,merge.go,actions_test.go}
│   ├── Makefile
│   └── README.md
├── bin/
│   └── cc-rich-shim             # bash wrapper exec'ing ~/.local/bin/cc-rich
├── tmux/tmux.conf               # +3 binds: R / C-r / M
├── claude/commands/cc-help.md   # +rows for new keybinds
└── docs/superpowers/specs/2026-05-01-cc-rich-design.md  # this file
```

The shim exists so the existing `tmux-claude-audit` helper-script-presence loop sees `~/bin/cc-rich` (consistent with cc-jump, cc-attention). The Go binary itself lives at `~/.local/bin/cc-rich`.

## Build & install

`cc-rich/Makefile`:

```make
build:
	go build -o ./bin/cc-rich ./cmd/cc-rich

install: build
	mkdir -p $$HOME/.local/bin
	cp ./bin/cc-rich $$HOME/.local/bin/cc-rich

test:
	go test ./...

install-shim:
	ln -sf $$HOME/dev/dotfiles/bin/cc-rich-shim $$HOME/bin/cc-rich
```

`install.sh` adds: `(cd cc-rich && make install install-shim)`.

## Module boundaries

| Module | Knows about | Doesn't know about |
|---|---|---|
| `sessiontree` | JSON, file paths, parent/child math | TUI, tmux, Glamour, Lipgloss |
| `view` | Models, Bubble Tea messages, Lipgloss styles, Glamour rendering | tmux command shape, file I/O (gets data via `sessiontree.Tree`) |
| `actions` | The `Runner` interface, file paths, command-line shape | TUI, Bubble Tea, JSON content (gets `Message` values from `sessiontree`) |
| `cmd/cc-rich/main.go` | Wires the three above + parses flags | Anything domain-specific (delegates) |

You can change Glamour styling without touching `actions`. You can change F1's mechanics without touching `view`. You can change the JSONL schema parser without touching anything else.

## Testing strategy

**Unit tests (Go, no TUI):**
- `sessiontree_test.go` — fixture JSONLs covering: linear conversation, single branch, multi-branch, partial-last-line, missing parent (orphan)
- `actions_test.go` — F1/F2/F4/Merge with mocked `Runner`; assert correct command-line shape and atomic file writes

**Integration tests:**
- `teatest` (Bubble Tea's testing harness) — drive the TUI through scripted keystrokes, capture screen output, snapshot-compare against golden files
- One golden test per major view: BranchList, Conversation, MergeComposer

**Live e2e (manual checklist):**
1. Open `cc-rich --pane #{pane_id}` against a real running Claude session — popup renders
2. Press `1` on a message → new tmux window opens with forked session
3. Continue work in fork → return to original via `cc-jump`
4. Re-open `cc-rich` → see both branches in tree
5. Merge selected messages → confirm `<cwd>/.cc-pending-prompt` exists with quoted blockquotes

**Audit + regression checks (added to existing `tmux-claude-test`):**
- `cc-rich` shim exists + executable
- `tmux-pane-header` knows about new keybinds (cc-help wired)
- Fixture-JSONL → `cc-rich --browse --headless --emit-html` round-trip produces stable output

## Out of scope (documented future work)

- Cross-machine session lineage (would need `cc-brain-sync` extension)
- LLM-summarized merge (Haiku call before merging)
- Editing past messages (append-only by design)
- Branch deletion / pruning UI (use existing `cc-prune`)
- Visual graph view for >2 levels deep (v1 is flat list)
- `Merge & Send` auto-injection (v1 stays explicit; v2 could add)

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| F1 spike fails on Claude Code's session-id validation | Medium — F1 collapses to F2-with-preamble | Spike-test in iter-1; fallback path already designed |
| Enterprise sandbox blocks `tmux new-window` from popup | Medium — fork actions fail with clear error | Documented; user adds permission rule once |
| Bubble Tea's mouse handling conflicts with tmux's mouse forwarding inside popups | Low — keyboard fallback always works | Test on iter-1; provide `--no-mouse` opt-out |
| 50k+ message sessions slow to render | Low — lazy-render ceiling | Already designed: last 200 turns + lazy hydration |

## Acceptance criteria

The feature is shippable when:

- `Ctrl-a R` opens a popup that renders the active pane's session richly within 200ms (TTI)
- F1 button creates a new tmux window with a forked Claude session whose history matches the original up to the picked message
- F2 button creates a new tmux window with a fresh Claude session whose first prompt is the picked message text
- F4 button appends a citation block to `~/.claude/buffer.md`
- Merge composer writes a citation blockquote to `<cwd>/.cc-pending-prompt`
- All unit tests pass; teatest goldens stable
- `tmux-claude-audit` includes `cc-rich` checks, all PASS
- `cc-help.md` documents the new bindings
- Manual e2e checklist passes end-to-end on the author's machine

## Implementation order (handed to writing-plans)

The implementation plan should sequence:

1. **Spike**: verify F1's prefix-copy approach against Claude Code's session-id validation. If fails, lock in F2-with-preamble fallback for F1.
2. **`sessiontree` package** (data layer; no TUI dependency)
3. **`actions` package with mock Runner** (side-effect layer; no TUI dependency)
4. **`cmd/cc-rich/main.go` minimal — just resolves session, prints to stdout** (no TUI yet; integration check)
5. **`view` package — `ConversationModel` first** (single-branch render via Glamour)
6. **`view` package — `BranchListModel`** (siblings discovered via `sessiontree.Siblings`)
7. **`view` package — `MergeComposerModel`**
8. **tmux glue** (3 binds in tmux.conf)
9. **Audit + tests integration**
10. **cc-help.md updates**
11. **Manual e2e checklist run**

Each step gets its own commit; checkpoints after steps 4, 7, and 11.
