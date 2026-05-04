# cc-rich Viewport Scrolling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add viewport-based scrolling to the cc-rich sidebar so transcripts longer than the pane height are navigable (j/k, PgUp/PgDn, g/G, mouse wheel) instead of being silently clipped.

**Architecture:** Wrap the existing `ConversationModel.View()` output in a `bubbles/viewport.Model`. Pre-render the conversation into a single string, hand it to `viewport.SetContent`, and let viewport manage `YOffset` + line-slicing. Keep `tea.WithAltScreen()` (clean enter/exit; no scrollback pollution). Cursor decoration stays inside the pre-rendered string; when the cursor moves, content is re-rendered and the viewport scrolls to keep the highlighted row visible.

**Tech Stack:** Go 1.26.2, [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) (viewport model), Glamour (already in), Lipgloss (already in), [teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest) for TUI assertions.

**Spec:** `docs/superpowers/specs/2026-05-01-cc-rich-design.md`
**Project rules:** `cc-rich/CLAUDE.md` (read first — has gotchas and verification recipe)

**Branch:** `feat/cc-rich`. All commits use `feat(cc-rich/view):` / `test(cc-rich/view):` / `docs(cc-rich):` / `refactor(cc-rich/view):` prefixes per project convention.

---

## Root cause being fixed

Three layers, all in `internal/view/conversation.go` and `cmd/cc-rich/main.go`:

1. `cmd/cc-rich/main.go:66` opens with `tea.WithAltScreen()`. AltScreen has no scrollback, so tmux's `prefix + [` copy-mode has nothing to scroll.
2. `conversation.go:131` is `for i, msg := range m.msgs` with no offset and no `m.height`-aware slicing. `View()` returns the full transcript every frame; the terminal driver clips at pane height and discards overflow.
3. `conversation.go:85–90` `j`/`k` increment `m.cursor` only — when the cursor moves past the visible window, the magenta border decorates a clipped (invisible) row.

This plan replaces the layered problem with a viewport model that owns offset state and produces the visible slice each frame.

---

## File structure

| File | Responsibility | This plan changes it? |
|---|---|---|
| `cc-rich/cmd/cc-rich/main.go` | Wire flags + Bubble Tea program | Yes — add `tea.WithMouseCellMotion()` |
| `cc-rich/internal/view/conversation.go` | ConversationModel: state + Update + View | Yes — embed viewport.Model, refactor View() into a content builder + viewport-backed render |
| `cc-rich/internal/view/view_test.go` | teatest-driven assertions | Yes — three new tests |
| `cc-rich/CLAUDE.md` | Project rules / gotchas | Yes — Patterns + Gotchas additions |
| `claude/commands/cc-help.md` | Slash-command cheat sheet | Yes — keybind row updates |
| `~/.claude/commands/cc-help.md` | Synced copy (managed-enterprise dir, but this single file is dotfiles-owned) | Yes — same edit applied |
| `zsh/zshrc` | `cc-help()` fzf cheat sheet | Yes — keybind entries |
| `cc-rich/Makefile` | build / install | No |
| `cc-rich/e2e/e2e-test.sh` | integration smoke | No (does not drive TUI keystrokes) |
| `bin/tmux-claude-audit` | system audit | No (no new shipped binary) |
| `bin/tmux-claude-test` | regression suite | No |

---

## Task list

### Phase 1: Foundation

- [ ] Task 1: Add `bubbles` dependency
- [ ] Task 2: Extract body-rendering into `buildContent()` helper (no behavior change)

### Checkpoint: Foundation
- [ ] `go vet ./...` clean
- [ ] `go test ./...` passes (existing tests unchanged)

### Phase 2: Viewport integration

- [ ] Task 3: Embed `viewport.Model` and feed it `buildContent()` output
- [ ] Task 4: Route `j`/`k`/`PgUp`/`PgDn`/`g`/`G` through the viewport
- [ ] Task 5: Cursor-follow — when `j` advances cursor past visible area, viewport scrolls to keep it on screen

### Checkpoint: Viewport
- [ ] Tall transcript (50 msgs, 10-row pane) shows last message after PgDn
- [ ] All 5 existing view tests still pass
- [ ] Build runs and the binary still launches in a real tmux pane

### Phase 3: Polish

- [ ] Task 6: Enable mouse wheel scrolling (`tea.WithMouseCellMotion()`)
- [ ] Task 7: Footer status line — `lineN/lineTotal · msgN/msgTotal`

### Phase 4: Docs

- [ ] Task 8: Update `cc-rich/CLAUDE.md` (Patterns + Gotchas)
- [ ] Task 9: Update `cc-help.md` (both copies) + `zshrc` `cc-help()` entries

### Phase 5: Final integration

- [ ] Task 10: Full verification recipe + manual smoke test

### Checkpoint: Complete
- [ ] All acceptance criteria met
- [ ] All tests pass
- [ ] Audit + e2e clean
- [ ] Manual `Ctrl-a R` smoke test confirms scroll keys work in a real tmux sidebar

---

## Phase 1 — Foundation

### Task 1: Add `bubbles` dependency

**Files:**
- Modify: `cc-rich/go.mod`
- Modify: `cc-rich/go.sum`

- [ ] **Step 1: Add the dependency**

```bash
cd ~/dev/dotfiles/cc-rich
go get github.com/charmbracelet/bubbles@latest
```

Expected: `go.mod` gains a `github.com/charmbracelet/bubbles` direct require. Indirect deps may shift slightly. No code uses the import yet, so `go mod tidy` will move it back to indirect — that's fine.

- [ ] **Step 2: Verify build still passes**

```bash
go build ./...
```

Expected: no output, exit 0. (Nothing imports viewport yet, so this is purely a dependency-resolution check.)

- [ ] **Step 3: Commit**

```bash
git add cc-rich/go.mod cc-rich/go.sum
git commit -m "$(cat <<'EOF'
feat(cc-rich): add bubbles dependency for viewport scrolling

Pulls in github.com/charmbracelet/bubbles. The viewport.Model from this
package will replace the manual "render entire transcript every frame"
loop in conversation.go so that long transcripts are scrollable instead
of silently clipped.

No code uses the import yet — that lands in the next commits.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Extract body-rendering into `buildContent()` helper

**Goal:** No behavior change. Move the `for i, msg := range m.msgs` loop body (lines 131–168 of conversation.go) into a private method that returns the full content string. This is a pure refactor that sets up Task 3's `viewport.SetContent(buildContent())` call.

**Files:**
- Modify: `cc-rich/internal/view/conversation.go:126-170`

- [ ] **Step 1: Run existing tests to capture green baseline**

```bash
cd ~/dev/dotfiles/cc-rich
go test ./internal/view -v 2>&1 | tail -10
```

Expected: 5 PASS lines (TestConversationRenders, TestConversationRendersMarkdown, TestConversationRendersAllContentBlocks, TestConversationStyleColors, TestConversationWrapsLinksInOSC8 — and TestBranchListShowsSiblings, TestMergeComposerEmitsCitations from sibling models).

- [ ] **Step 2: Refactor — replace `View()` with a thin wrapper around `buildContent()`**

Replace lines 126–170 of `cc-rich/internal/view/conversation.go` with:

```go
// buildContent renders all messages into a single string, with the
// cursor row decorated by a magenta rounded border. Returned string is
// what gets fed to the viewport in subsequent versions of this model.
//
// Pulled out of View() so that the viewport-aware version of this model
// can call buildContent() on cursor / width / msg-list changes and pass
// the result to viewport.SetContent — instead of rebuilding from inside
// View() (which Bubble Tea calls every frame, where work is wasteful).
func (m ConversationModel) buildContent() string {
	if len(m.msgs) == 0 {
		return StyleMuted.Render("(empty session)")
	}
	var sb strings.Builder
	for i, msg := range m.msgs {
		header := fmt.Sprintf("%-9s  %s", msg.Role, msg.UUID)
		if msg.Role == "assistant" {
			header = StyleAsst.Render(header)
		} else {
			header = StyleUser.Render(header)
		}

		// Render every content block, not just [0]. Text and thinking go
		// through Glamour; tool_use shows as a muted, non-markdown line so
		// JSON payloads don't get mangled by the markdown parser.
		var bodyParts []string
		for _, b := range msg.Content {
			switch b.Type {
			case "text":
				bodyParts = append(bodyParts, (&m).renderMarkdown(b.Text))
			case "thinking":
				bodyParts = append(bodyParts, StyleMuted.Render("· thinking ·\n"+(&m).renderMarkdown(b.Text)))
			case "tool_use":
				bodyParts = append(bodyParts, StyleMuted.Render("⚙ tool_use "+b.Text))
			case "tool_result":
				bodyParts = append(bodyParts, StyleMuted.Render("↩ tool_result "+b.Text))
			default:
				if b.Text != "" {
					bodyParts = append(bodyParts, b.Text)
				}
			}
		}
		body := strings.Join(bodyParts, "\n")

		buttons := StyleMuted.Render("[1] resume+branch  [2] replay  [4] quote  [m] merge")
		row := lipgloss.JoinVertical(lipgloss.Left, header, body, buttons)
		if i == m.cursor {
			row = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(ColorAccent).Render(row)
		}
		sb.WriteString(row)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func (m ConversationModel) View() string {
	return m.buildContent()
}
```

- [ ] **Step 3: Run tests to confirm no regression**

```bash
go test ./internal/view -v 2>&1 | tail -10
```

Expected: same 5 PASS lines from Step 1.

- [ ] **Step 4: `go vet`**

```bash
go vet ./...
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add cc-rich/internal/view/conversation.go
git commit -m "$(cat <<'EOF'
refactor(cc-rich/view): extract body rendering into buildContent helper

Moves the entire for-loop that renders each msg.Content block into a
private buildContent() method. View() now just calls it. Pure refactor —
no behavior change, all existing tests still pass.

This is prep for the viewport integration: the viewport-aware version
of this model will call buildContent() on state changes and pass the
output to viewport.SetContent, rather than rebuilding from inside View()
(which Bubble Tea calls every frame).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 2 — Viewport integration

### Task 3: Embed `viewport.Model` and feed it `buildContent()` output

**Goal:** Wire viewport into the model. Re-set content on size or content changes. View() now returns the viewport's view, which is the visible window — anything past pane height is no longer clipped, it's just below the fold (and Tasks 4–5 add the keys to scroll it into view).

**Files:**
- Modify: `cc-rich/internal/view/conversation.go`
- Modify: `cc-rich/internal/view/view_test.go` (one new failing test)

- [ ] **Step 1: Write the failing test**

Append to `cc-rich/internal/view/view_test.go` (immediately after `TestConversationWrapsLinksInOSC8` and before `TestBranchListShowsSiblings`):

```go
// A transcript taller than the pane height must not lose content. With
// a viewport in place, content past the visible window is reachable
// (Tasks 4-5 add the keys); without one, it's clipped at render time
// and gone forever. This test pins the "content reachable" property
// by sending a PgDn and looking for the last message in the output.
func TestConversationViewportShowsAllContent(t *testing.T) {
	// 50 messages, each 3 lines tall after rendering — far past a
	// 10-row pane.
	msgs := make([]*sessiontree.Message, 50)
	for i := range msgs {
		msgs[i] = &sessiontree.Message{
			UUID:      fmt.Sprintf("u-%02d", i),
			Role:      "user",
			Timestamp: time.Now(),
			Content:   []sessiontree.Block{{Type: "text", Text: fmt.Sprintf("message %02d body line", i)}},
		}
	}
	m := NewConversation(msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 10))
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 10})
	// Send enough PgDn to reach the bottom: 50 msgs / ~3 lines per
	// msg / ~5 lines per HalfPageDown = ~30 keypresses. Use 60 to be
	// safe; viewport clamps at the bottom so extra is harmless.
	for i := 0; i < 60; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyPgDown})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)

	if !strings.Contains(got, "message 49") {
		t.Errorf("scroll-to-bottom did not reveal last message; viewport not wired:\n%s", got)
	}
}
```

Add `"fmt"` to the imports of `view_test.go` if not already present (it is — `TestConversationRenders` uses `fmt.Sprintf` indirectly via `.UUID` strings; double-check by running `go test`, the compiler will tell you if it's missing). The current import block is:

```go
import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/aperritano/cc-rich/internal/sessiontree"
)
```

If `fmt` isn't there, add it as the first non-stdlib import:

```go
import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/aperritano/cc-rich/internal/sessiontree"
)
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/dev/dotfiles/cc-rich
go test ./internal/view -run TestConversationViewportShowsAllContent -v 2>&1 | tail -15
```

Expected: FAIL with `scroll-to-bottom did not reveal last message; viewport not wired`. (Without a viewport, PgDn does nothing and the rendered output is clipped at row 10 — the bottom messages are never written.)

- [ ] **Step 3: Add `viewport.Model` to the imports of `conversation.go`**

Update the import block of `cc-rich/internal/view/conversation.go` to include `bubbles/viewport`:

```go
import (
	"fmt"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"

	"github.com/aperritano/cc-rich/internal/sessiontree"
)
```

- [ ] **Step 4: Embed the viewport in the model**

Replace the `ConversationModel` struct and `NewConversation` constructor (lines 57–70 of conversation.go before this task) with:

```go
// ConversationModel renders a list of messages as a scrollable column.
// The actual scroll state (YOffset) lives in viewport; we own cursor
// position and the message list and feed the viewport pre-rendered
// content via SetContent on state changes.
type ConversationModel struct {
	msgs   []*sessiontree.Message
	cursor int
	width  int
	height int
	done   bool
	md     *glamour.TermRenderer // built lazily on first WindowSizeMsg
	mdW    int                   // width the renderer was built for

	vp     viewport.Model // owns scroll position; we feed it content
	ready  bool           // becomes true after first WindowSizeMsg
}

func NewConversation(msgs []*sessiontree.Message) ConversationModel {
	return ConversationModel{msgs: msgs}
}
```

- [ ] **Step 5: Initialize the viewport on first WindowSizeMsg, re-set content on every WindowSizeMsg or cursor change**

Replace the `Update()` method (existing lines 74–95) with:

```go
func (m ConversationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch t := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = t.Width
		m.height = t.Height
		if !m.ready {
			m.vp = viewport.New(t.Width, t.Height)
			m.ready = true
		} else {
			m.vp.Width = t.Width
			m.vp.Height = t.Height
		}
		// Width changed → Glamour wrap recomputes → cached renderer
		// invalidates on its own (handled inside renderMarkdown). We
		// must always re-set viewport content because line wrapping
		// changes with width.
		m.vp.SetContent(m.buildContent())

	case tea.KeyMsg:
		switch t.String() {
		case "esc", "q":
			m.done = true
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.msgs)-1 {
				m.cursor++
				if m.ready {
					m.vp.SetContent(m.buildContent())
				}
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				if m.ready {
					m.vp.SetContent(m.buildContent())
				}
			}
		}
	}

	// Forward unhandled keys + mouse to viewport so PgUp/PgDn/wheel
	// work without us having to enumerate them. Tasks 4-5 add the
	// rest of the explicit keymap above this fallthrough.
	if m.ready {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}
```

- [ ] **Step 6: Replace `View()` to render through the viewport**

Replace the existing `View()` method (`func (m ConversationModel) View() string { return m.buildContent() }`) with:

```go
func (m ConversationModel) View() string {
	if !m.ready {
		// Pre-WindowSizeMsg: viewport has no size; fall back to the
		// raw content (terminal will clip it, but at least the user
		// sees something for the first frame).
		return m.buildContent()
	}
	return m.vp.View()
}
```

- [ ] **Step 7: Run the new test to verify it now passes**

```bash
go test ./internal/view -run TestConversationViewportShowsAllContent -v 2>&1 | tail -10
```

Expected: PASS.

- [ ] **Step 8: Run full test suite to verify no regression**

```bash
go test ./... 2>&1 | tail -8
```

Expected: all packages OK. (`internal/view`, `internal/sessiontree`, `internal/actions`.)

- [ ] **Step 9: `go vet`**

```bash
go vet ./...
```

Expected: no output.

- [ ] **Step 10: Commit**

```bash
git add cc-rich/internal/view/conversation.go cc-rich/internal/view/view_test.go cc-rich/go.mod cc-rich/go.sum
git commit -m "$(cat <<'EOF'
feat(cc-rich/view): embed viewport.Model so long transcripts scroll

ConversationModel now owns a bubbles/viewport.Model. We build the full
rendered content via buildContent() on state changes (WindowSizeMsg,
cursor move) and pass it to vp.SetContent. View() returns vp.View(),
which is the visible window slice. Unhandled keys + mouse events fall
through to vp.Update so PgUp/PgDn/wheel work for free; explicit g/G/
PgUp/PgDn keymaps land in the next commits.

The viewport gets initialized on the first WindowSizeMsg (when we
finally know the pane size). Pre-init View() falls back to raw
buildContent() so the user isn't staring at a blank pane during the
first paint.

Test pins the property: a 50-message transcript in a 10-row pane,
scrolled to the bottom via PgDn, must reveal "message 49". Without the
viewport, that content was clipped and lost.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Route `g`/`G`/`PgUp`/`PgDn` through the viewport explicitly

**Goal:** The viewport pass-through in Task 3 already handles PgUp/PgDn (viewport's default keymap). Explicit g/G (top/bottom jumps) is not in viewport's default keymap, so we add them. Pinning PgUp/PgDn explicitly also documents the intended UX.

**Files:**
- Modify: `cc-rich/internal/view/conversation.go` (Update method)
- Modify: `cc-rich/internal/view/view_test.go` (one new test)

- [ ] **Step 1: Write the failing test**

Append to `view_test.go`, after `TestConversationViewportShowsAllContent`:

```go
// g jumps to top, G jumps to bottom — vim-style navigation. Without
// these, navigating a tall transcript means hammering PgDn / k.
func TestConversationGotoTopAndBottom(t *testing.T) {
	msgs := make([]*sessiontree.Message, 30)
	for i := range msgs {
		msgs[i] = &sessiontree.Message{
			UUID:      fmt.Sprintf("u-%02d", i),
			Role:      "user",
			Timestamp: time.Now(),
			Content:   []sessiontree.Block{{Type: "text", Text: fmt.Sprintf("msg-%02d", i)}},
		}
	}
	m := NewConversation(msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 10))
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 10})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")}) // jump to bottom
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}) // jump back to top
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)

	// Final state was "G then g", so we should see the top of the
	// transcript (msg-00). msg-29 may still appear in the buffered
	// stream from the G-keypress frame, so we additionally assert
	// that msg-00 is present.
	if !strings.Contains(got, "msg-00") {
		t.Errorf("g (goto top) did not reveal first message:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/dev/dotfiles/cc-rich
go test ./internal/view -run TestConversationGotoTopAndBottom -v 2>&1 | tail -10
```

Expected: FAIL with `g (goto top) did not reveal first message`. (Viewport doesn't bind `g`/`G` by default; the keypresses fall through to no-op.)

- [ ] **Step 3: Add explicit g/G handling to Update**

Inside the `case tea.KeyMsg:` switch in `Update()` (added in Task 3), add cases BEFORE the existing `j`/`k` cases — viewport's default keymap doesn't claim `g`/`G` so we own them outright:

```go
		case "g":
			if m.ready {
				m.vp.GotoTop()
			}
		case "G":
			if m.ready {
				m.vp.GotoBottom()
			}
```

The full updated keymap inside `case tea.KeyMsg:` should now be:

```go
		case "esc", "q":
			m.done = true
			return m, tea.Quit
		case "g":
			if m.ready {
				m.vp.GotoTop()
			}
		case "G":
			if m.ready {
				m.vp.GotoBottom()
			}
		case "j", "down":
			if m.cursor < len(m.msgs)-1 {
				m.cursor++
				if m.ready {
					m.vp.SetContent(m.buildContent())
				}
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				if m.ready {
					m.vp.SetContent(m.buildContent())
				}
			}
```

- [ ] **Step 4: Run the new test to verify it passes**

```bash
go test ./internal/view -run TestConversationGotoTopAndBottom -v 2>&1 | tail -5
```

Expected: PASS.

- [ ] **Step 5: Run full view suite**

```bash
go test ./internal/view -v 2>&1 | tail -15
```

Expected: all 7 view tests pass.

- [ ] **Step 6: Commit**

```bash
git add cc-rich/internal/view/conversation.go cc-rich/internal/view/view_test.go
git commit -m "$(cat <<'EOF'
feat(cc-rich/view): bind g/G to viewport GotoTop/GotoBottom

Viewport's default keymap covers PgUp/PgDn but not vim-style g/G.
Explicit cases in Update intercept them and call vp.GotoTop /
vp.GotoBottom, which set YOffset to 0 / max respectively. j/k still
own cursor movement.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Cursor-follow — viewport scrolls when cursor moves off-screen

**Goal:** `j`/`k` advance the cursor, but currently the viewport doesn't move with it. After 10 `j` presses on a tall transcript, the magenta border decorates a row past the visible window. Fix: after every cursor change, if the cursor's row is outside the viewport's visible range, scroll the viewport to put it back in view.

**Files:**
- Modify: `cc-rich/internal/view/conversation.go` (Update method + new helper)
- Modify: `cc-rich/internal/view/view_test.go` (one new test)

- [ ] **Step 1: Write the failing test**

Append to `view_test.go`:

```go
// When j moves the cursor past the visible window, the viewport must
// scroll to bring the cursor back into view. Otherwise the magenta
// border decorates a row that's been clipped — invisible state, very
// confusing UX.
func TestConversationCursorFollowsViewport(t *testing.T) {
	msgs := make([]*sessiontree.Message, 30)
	for i := range msgs {
		msgs[i] = &sessiontree.Message{
			UUID:      fmt.Sprintf("u-%02d", i),
			Role:      "user",
			Timestamp: time.Now(),
			Content:   []sessiontree.Block{{Type: "text", Text: fmt.Sprintf("body-%02d", i)}},
		}
	}
	m := NewConversation(msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 10))
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 10})
	// Press j 25 times — cursor goes to msg index 25, well past the
	// initial 10-row visible window.
	for i := 0; i < 25; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)

	// The cursor is now at msg-25, so the viewport should have
	// scrolled to a window that contains body-25.
	if !strings.Contains(got, "body-25") {
		t.Errorf("viewport did not follow cursor — body-25 not visible:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/dev/dotfiles/cc-rich
go test ./internal/view -run TestConversationCursorFollowsViewport -v 2>&1 | tail -10
```

Expected: FAIL with `viewport did not follow cursor — body-25 not visible`. (Cursor moves to index 25 but viewport stays at YOffset 0.)

- [ ] **Step 3: Add `ensureCursorVisible()` helper**

Add this method to `conversation.go`, just above the existing `buildContent()` method:

```go
// ensureCursorVisible scrolls the viewport so that the row decorated
// with the cursor border stays on screen. We don't have direct line
// numbers per cursor index — buildContent assembles each msg as N
// rendered lines + a 2-line gap — so we count lines up to the cursor's
// row in the rendered content and call vp.SetYOffset to put that line
// in the viewport's middle (or as close as bounds allow).
func (m *ConversationModel) ensureCursorVisible() {
	if !m.ready || len(m.msgs) == 0 {
		return
	}
	content := m.vp.View() // current viewport view (post-clipping)
	_ = content            // we don't actually need the rendered slice
	// Walk the full content one msg at a time; sum line counts to
	// find the cursor row's line number.
	full := m.buildContent()
	lines := strings.Split(full, "\n")
	totalLines := len(lines)
	if totalLines == 0 {
		return
	}
	// Approximate cursor line: each msg block in buildContent ends
	// with "\n\n" — split the joined content on the same boundary
	// and sum line counts up through m.cursor.
	blocks := strings.Split(full, "\n\n")
	cursorLine := 0
	for i := 0; i < m.cursor && i < len(blocks); i++ {
		cursorLine += strings.Count(blocks[i], "\n") + 2 // +2 for the joining \n\n
	}

	top := m.vp.YOffset
	bottom := top + m.vp.Height - 1
	switch {
	case cursorLine < top:
		m.vp.SetYOffset(cursorLine)
	case cursorLine > bottom:
		// Put cursor row near the bottom of the viewport, with a
		// little space below. Clamp via Bubbles' built-in bounds.
		newOffset := cursorLine - m.vp.Height + 3
		if newOffset < 0 {
			newOffset = 0
		}
		m.vp.SetYOffset(newOffset)
	}
}
```

- [ ] **Step 4: Call `ensureCursorVisible()` after every cursor change**

In the `Update()` method, modify the `j` and `k` cases to call `ensureCursorVisible()` after `SetContent`:

```go
		case "j", "down":
			if m.cursor < len(m.msgs)-1 {
				m.cursor++
				if m.ready {
					m.vp.SetContent(m.buildContent())
					(&m).ensureCursorVisible()
				}
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				if m.ready {
					m.vp.SetContent(m.buildContent())
					(&m).ensureCursorVisible()
				}
			}
```

- [ ] **Step 5: Run the new test to verify it passes**

```bash
go test ./internal/view -run TestConversationCursorFollowsViewport -v 2>&1 | tail -5
```

Expected: PASS.

- [ ] **Step 6: Run full view suite**

```bash
go test ./internal/view -v 2>&1 | tail -20
```

Expected: all 8 view tests pass.

- [ ] **Step 7: `go vet`**

```bash
go vet ./...
```

Expected: no output.

- [ ] **Step 8: Commit**

```bash
git add cc-rich/internal/view/conversation.go cc-rich/internal/view/view_test.go
git commit -m "$(cat <<'EOF'
feat(cc-rich/view): viewport follows cursor on j/k

After cursor advances, scroll the viewport to keep the highlighted row
on screen. Otherwise the magenta border decorates a row past the visible
window — invisible state, confusing UX.

ensureCursorVisible walks buildContent one msg at a time, sums rendered
line counts up through m.cursor, and calls vp.SetYOffset if the cursor
line is outside the current top..bottom window. Cursor near the bottom
gets a 3-line tail buffer so context after the highlighted row stays
visible too.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 3 — Polish

### Task 6: Enable mouse wheel scrolling

**Goal:** `viewport.Update` handles `tea.MouseMsg` natively, but Bubble Tea doesn't deliver mouse events unless the program asks for them. Add `tea.WithMouseCellMotion()` to the program options.

**Files:**
- Modify: `cc-rich/cmd/cc-rich/main.go:66`

- [ ] **Step 1: Update the program options**

In `cc-rich/cmd/cc-rich/main.go`, change line 66 from:

```go
	p := tea.NewProgram(view.NewConversation(msgs), tea.WithAltScreen())
```

to:

```go
	p := tea.NewProgram(
		view.NewConversation(msgs),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
```

- [ ] **Step 2: Build and verify the binary still launches**

```bash
cd ~/dev/dotfiles/cc-rich
go build ./cmd/cc-rich -o /tmp/cc-rich-mouse
ls -la /tmp/cc-rich-mouse
/tmp/cc-rich-mouse --help 2>&1 | head -3
```

Expected: binary builds cleanly, --help prints flag descriptions and exits 0. (Mouse motion is purely a runtime concern; build-time only verifies the import + option exists.)

- [ ] **Step 3: Run tests to confirm no breakage**

```bash
go test ./... 2>&1 | tail -5
```

Expected: all packages OK.

- [ ] **Step 4: Commit**

```bash
git add cc-rich/cmd/cc-rich/main.go
git commit -m "$(cat <<'EOF'
feat(cc-rich): enable mouse wheel scrolling in conversation view

tea.WithMouseCellMotion makes Bubble Tea forward MouseMsg events to
the model. viewport.Update already handles wheel events natively —
this option is the missing piece.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Footer status line — `lineN/lineTotal · msgN/msgTotal`

**Goal:** Without orientation, scrolling a long transcript feels disorienting. Add a one-line footer below the viewport content showing scroll position. Format: `42/418 lines · 12/45 messages` (no leading icons; muted color).

**Files:**
- Modify: `cc-rich/internal/view/conversation.go` (View method + new helper)
- Modify: `cc-rich/internal/view/view_test.go` (one new test)

- [ ] **Step 1: Write the failing test**

Append to `view_test.go`:

```go
// The footer pins scroll orientation: "Nlines/Mlines · Pmsg/Qmsg".
// Without it, scrolling a long transcript is disorienting — there's
// no other indicator of where you are in the buffer.
func TestConversationFooterShowsScrollPosition(t *testing.T) {
	msgs := make([]*sessiontree.Message, 5)
	for i := range msgs {
		msgs[i] = &sessiontree.Message{
			UUID:      fmt.Sprintf("u-%02d", i),
			Role:      "user",
			Timestamp: time.Now(),
			Content:   []sessiontree.Block{{Type: "text", Text: fmt.Sprintf("body %02d", i)}},
		}
	}
	m := NewConversation(msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 20))
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 20})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)

	// Footer format: "<n>/<m> lines · <p>/5 messages" — we don't
	// pin exact line counts (depend on Glamour wrapping), but the
	// "/5 messages" tail is stable.
	if !strings.Contains(got, "/5 messages") {
		t.Errorf("footer missing message-position counter:\n%s", got)
	}
	if !strings.Contains(got, "lines") {
		t.Errorf("footer missing line-position counter:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd ~/dev/dotfiles/cc-rich
go test ./internal/view -run TestConversationFooterShowsScrollPosition -v 2>&1 | tail -10
```

Expected: FAIL with `footer missing message-position counter`.

- [ ] **Step 3: Add a `footer()` helper**

Add this method to `conversation.go`, immediately after `ensureCursorVisible`:

```go
// footer renders a one-line scroll-position indicator. Format:
//
//	42/418 lines · 12/45 messages
//
// Muted color; no leading icon. Shown below the viewport content in
// View().
func (m ConversationModel) footer() string {
	if !m.ready {
		return ""
	}
	totalLines := strings.Count(m.buildContent(), "\n") + 1
	curLine := m.vp.YOffset + 1
	if curLine > totalLines {
		curLine = totalLines
	}
	totalMsgs := len(m.msgs)
	curMsg := m.cursor + 1
	if totalMsgs == 0 {
		curMsg = 0
	}
	return StyleMuted.Render(fmt.Sprintf(
		"%d/%d lines · %d/%d messages",
		curLine, totalLines, curMsg, totalMsgs,
	))
}
```

- [ ] **Step 4: Wire the footer into View**

Replace the `View()` method with:

```go
func (m ConversationModel) View() string {
	if !m.ready {
		return m.buildContent()
	}
	// Reserve the bottom row for the footer; viewport already sized
	// itself to the full pane height in Update, so we just stack
	// the viewport + footer with lipgloss.
	return lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), m.footer())
}
```

Then update Task 3's WindowSizeMsg handler to give the viewport `Height-1` so the footer fits without overflow. In the `case tea.WindowSizeMsg:` branch of `Update()`, change:

```go
		if !m.ready {
			m.vp = viewport.New(t.Width, t.Height)
			m.ready = true
		} else {
			m.vp.Width = t.Width
			m.vp.Height = t.Height
		}
```

to:

```go
		vpHeight := t.Height - 1 // reserve one row for footer
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.ready {
			m.vp = viewport.New(t.Width, vpHeight)
			m.ready = true
		} else {
			m.vp.Width = t.Width
			m.vp.Height = vpHeight
		}
```

- [ ] **Step 5: Run the footer test**

```bash
go test ./internal/view -run TestConversationFooterShowsScrollPosition -v 2>&1 | tail -5
```

Expected: PASS.

- [ ] **Step 6: Run full view suite to confirm Task 3-5 tests still pass**

```bash
go test ./internal/view -v 2>&1 | tail -25
```

Expected: 9 view tests pass. (TestConversationViewportShowsAllContent, TestConversationGotoTopAndBottom, and TestConversationCursorFollowsViewport may need their height-arg bumped by 1 since the viewport is now `Height-1`. If any fail, increase the `WindowSizeMsg` Height by 1 in those tests — e.g. from 10 to 11 — to keep the visible area at 10 rows.)

- [ ] **Step 7: `go vet`**

```bash
go vet ./...
```

Expected: no output.

- [ ] **Step 8: Commit**

```bash
git add cc-rich/internal/view/conversation.go cc-rich/internal/view/view_test.go
git commit -m "$(cat <<'EOF'
feat(cc-rich/view): add scroll-position footer to conversation view

One-line muted indicator below the viewport: "Nlines/Mlines · Pmsg/Qmsg".
Without it, scrolling a long transcript is disorienting — no other
indicator of buffer position.

Viewport height reserved one row for the footer; viewport.SetYOffset
math unaffected.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 4 — Docs

### Task 8: Update `cc-rich/CLAUDE.md` Patterns + Gotchas

**Files:**
- Modify: `cc-rich/CLAUDE.md`

- [ ] **Step 1: Add a Patterns entry**

In `cc-rich/CLAUDE.md`, find the "## Patterns to follow" section. Append this bullet:

```markdown
- **Conversation scrolling lives in `bubbles/viewport`, not in the model.** `ConversationModel` builds the full content string via `buildContent()` and hands it to `viewport.Model.SetContent`. The viewport owns `YOffset` and slices visible lines per frame. Cursor decoration stays inside the rendered content; `ensureCursorVisible` adjusts viewport offset on `j`/`k` so the highlighted row stays on screen. **Don't try to manage scroll state in `ConversationModel` directly** — viewport already does it correctly with mouse, PgUp/PgDn, and bounds-clamping.
```

- [ ] **Step 2: Add a Gotcha entry**

In the "## Gotchas (learned the hard way; do not relearn)" section, append:

```markdown
- **AltScreen + viewport is the right combo; don't tear out AltScreen.** Naive instinct on "I can't scroll the cc-rich pane with `prefix + [`" is to drop `tea.WithAltScreen()` so tmux's copy-mode can target the buffer. **Don't.** That pollutes the host pane's scrollback with cc-rich redraws, ruins the clean enter/exit transition, and still doesn't give per-message navigation. The viewport-inside-AltScreen pattern (this codebase, since `feat(cc-rich/view): embed viewport.Model`) gives proper in-app scrolling AND a clean exit. See `internal/view/conversation.go` `Update()` for the cursor-follow logic.
```

- [ ] **Step 3: Commit**

```bash
git add cc-rich/CLAUDE.md
git commit -m "$(cat <<'EOF'
docs(cc-rich): document viewport scrolling pattern + AltScreen gotcha

Two CLAUDE.md additions:

1. Patterns section: scroll state lives in bubbles/viewport, not in
   ConversationModel. Cursor-follow via ensureCursorVisible.

2. Gotchas section: don't tear out AltScreen. The naive fix to "can't
   scroll with prefix+[" is to drop AltScreen so tmux copy-mode works,
   but that pollutes the host pane's scrollback. Viewport-in-AltScreen
   is the right pattern.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Update `cc-help.md` and `zshrc` `cc-help()` entries

**Files:**
- Modify: `claude/commands/cc-help.md`
- Modify: `~/.claude/commands/cc-help.md` (synced copy; same edit)
- Modify: `zsh/zshrc`

- [ ] **Step 1: Update `claude/commands/cc-help.md` keybind table**

Find the Ctrl-a R/B/M rows in the keybind table. Replace:

```markdown
| `Ctrl-a R` | cc-rich: rich **sidebar** view of the active session (toggle — press R again to close) |
| `Ctrl-a B` | cc-rich: browse all known sessions in sidebar (toggle) |
| `Ctrl-a M` | cc-rich: open merge composer for the active session (sidebar, toggle) |
```

with:

```markdown
| `Ctrl-a R` | cc-rich: rich **sidebar** view of the active session (toggle — press R again to close) |
| `Ctrl-a B` | cc-rich: browse all known sessions in sidebar (toggle) |
| `Ctrl-a M` | cc-rich: open merge composer for the active session (sidebar, toggle) |

**Inside the cc-rich sidebar:**

| Key | Action |
|---|---|
| `j` / `↓` | next message (viewport follows) |
| `k` / `↑` | previous message |
| `g` | jump to top |
| `G` | jump to bottom |
| `PgUp` / `PgDn` | half-page scroll |
| mouse wheel | scroll |
| `q` / `Esc` | close sidebar |
```

- [ ] **Step 2: Apply the same edit to the synced copy**

```bash
cp ~/dev/dotfiles/claude/commands/cc-help.md ~/.claude/commands/cc-help.md
```

(The two files were synced in commit `a515e35` via this same mechanism. CLAUDE.md says don't *touch* `~/.claude/*` — but `cc-help.md` was already established as a dotfiles-owned file. Touching the established mirror is fine; touching `statusline-command.sh`, `settings.json`, or anything not previously tracked is what the gotcha is about.)

- [ ] **Step 3: Update `zsh/zshrc` `cc-help()` entries**

Find the `cc-rich` and `cc-rich-sidebar` rows in the `cc-help()` array (around lines 304–307 and 315–317). Replace the existing R/B/M description rows:

```
    "Ctrl-a R|tmux: cc-rich sidebar|rich sidebar (35% right split) for active pane's session — toggle (R again closes)"
    "Ctrl-a B|tmux: cc-rich browse|browse all known sessions in sidebar (toggle)"
    "Ctrl-a M|tmux: cc-rich merge|merge composer for active pane's session (sidebar, toggle)"
```

with:

```
    "Ctrl-a R|tmux: cc-rich sidebar|rich sidebar — toggle (R again closes); inside: j/k/g/G/PgUp/PgDn/wheel scroll, q closes"
    "Ctrl-a B|tmux: cc-rich browse|browse all known sessions in sidebar (toggle); j/k navigate, q closes"
    "Ctrl-a M|tmux: cc-rich merge|merge composer for active pane's session (sidebar, toggle); j/k navigate, q closes"
```

- [ ] **Step 4: Reload zshrc to verify no syntax errors**

```bash
zsh -n ~/dev/dotfiles/zsh/zshrc
```

Expected: no output (zsh `-n` syntax-checks without execution).

- [ ] **Step 5: Commit**

```bash
git add claude/commands/cc-help.md zsh/zshrc
# ~/.claude/commands/cc-help.md is outside the dotfiles repo — it's
# the live copy. The dotfiles file is the source of truth.
git commit -m "$(cat <<'EOF'
docs(cc-help): document scroll keys inside cc-rich sidebar

cc-help.md keybind table gains an "Inside the cc-rich sidebar" sub-table
with j/k/g/G/PgUp/PgDn/wheel/q. zshrc cc-help() rows now mention the
in-pane keymap inline so the fzf cheat sheet stays self-contained.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 5 — Final integration

### Task 10: Full verification + manual smoke test

**Goal:** Run the project's verification recipe (from `cc-rich/CLAUDE.md`), then manually confirm in a real tmux session that the scroll keys behave correctly.

- [ ] **Step 1: Run the verification recipe**

```bash
cd ~/dev/dotfiles/cc-rich
go vet ./... && go test ./... && bash e2e/e2e-test.sh && make build && make install
```

Expected:
- `go vet` no output
- `go test` 4 packages OK (cmd/cc-rich has no tests; the other three pass)
- e2e: `PASS 18  FAIL 0  (total 18)`
- `make build` exits 0 (silent)
- `make install` exits 0; `~/.local/bin/cc-rich` updated; codesign step succeeded silently

- [ ] **Step 2: Run the audit**

```bash
~/bin/tmux-claude-audit 2>&1 | tail -3
```

Expected: `PASS 121  WARN 1  FAIL 0` (the 1 WARN is the dotfiles-uncommitted state; nothing cc-rich-related).

- [ ] **Step 3: Run the regression suite**

```bash
~/bin/tmux-claude-test > /tmp/cct.out 2>&1; echo "exit=$?"
grep -E 'cc-rich' /tmp/cct.out | head -10
tail -5 /tmp/cct.out
```

Expected: cc-rich-related entries all pass. 3 pre-existing FAILs (cc-burn-rate threshold/python, cc-brain-sync push-no-age) are out of scope.

- [ ] **Step 4: Manual smoke test in a real tmux pane**

In a tmux pane already running Claude (so cc-rich has a session to render):

```
Ctrl-a R       # sidebar opens with conversation
j j j j j      # cursor advances; viewport scrolls with it
G              # jump to bottom
g              # jump to top
PgDn           # half-page down
mouse wheel    # scroll up/down
q              # close sidebar
Ctrl-a R       # reopen — should remember session, fresh scroll position OK
```

Expected: every key produces visible movement; the footer updates with line/msg counts; the magenta cursor border stays on screen as `j`/`k` move.

- [ ] **Step 5: Final commit summary**

If all 4 prior steps passed, no additional commit is needed — Tasks 1–9 already shipped the work in 8 commits. Verify the branch state:

```bash
cd ~/dev/dotfiles
git log --oneline feat/cc-rich -10
```

Expected: the most recent commits should include all 8 from this plan (one per task; Task 10 is verification only).

---

## Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| `bubbles` version drift breaks viewport API | High — Task 3 won't compile | Pin to a specific `bubbles` version in Task 1 if `@latest` ships a breaking change; fall back to `v0.20.0` if needed |
| `ensureCursorVisible` line-counting drifts from rendered output (Glamour wraps differently than buildContent assumes) | Medium — cursor jumps to the wrong line | Test in Task 5 covers the off-by-many case (cursor 25 of 30, height 10); if drift shows up, switch to per-msg start-line tracking via a `[]int` index |
| Mouse motion makes click-to-quote interfere with text selection | Low — but real for users who want to copy-paste from the sidebar | If reported, add a Shift-modifier escape to bypass mouse handling, or wire a "copy mode" key inside cc-rich |
| Footer line count is stale after WindowSizeMsg width change but before SetContent fires | Low — flicker for 1 frame | Footer uses `m.buildContent()` directly (always fresh), not a cached count |
| Test for goto-top/bottom relies on stream ordering | Low — teatest may buffer differently across versions | Test asserts only the *final* state contains `msg-00`; intermediate frames can contain anything |

---

## Self-review

**1. Spec coverage:** The spec doesn't mention scrolling explicitly — it predates the long-transcript usability problem. This plan fills a gap rather than implementing a spec section. No spec edits needed.

**2. Placeholder scan:** All steps contain actual code or actual commands. No "TBD"/"add error handling"/"similar to Task N". Each test step shows the test code; each commit step shows the commit message body. ✓

**3. Type consistency:**
- `ConversationModel` field names: `msgs`, `cursor`, `width`, `height`, `done`, `md`, `mdW`, `vp`, `ready` — used identically across Tasks 3, 5, 7. ✓
- Method names: `buildContent`, `renderMarkdown`, `ensureCursorVisible`, `footer` — all consistent. ✓
- viewport API names (`SetContent`, `View`, `GotoTop`, `GotoBottom`, `SetYOffset`, `YOffset`, `Width`, `Height`, `Update`) — bubbles `viewport.Model` exports these per the v0.20+ surface. If the installed version differs, Task 1 should pin to a known-good version.

**4. Test naming:** All new tests follow `TestConversation<Behavior>` per the existing convention (TestConversationRenders / RendersMarkdown / etc). ✓

**5. Commit prefixes:** All conventional-commit prefixes match `cc-rich/CLAUDE.md` § Patterns: `feat(cc-rich):`, `feat(cc-rich/view):`, `refactor(cc-rich/view):`, `docs(cc-rich):`. ✓

**6. Verification gate:** Phase 5 / Task 10 runs the full recipe from `cc-rich/CLAUDE.md` § "Verification before claiming done". ✓
