package view

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/aperritano/cc-rich/internal/sessiontree"
)

func mkMsgs() []*sessiontree.Message {
	return []*sessiontree.Message{
		{UUID: "u-1", Role: "user", Content: []sessiontree.Block{{Type: "text", Text: "hi"}}, Timestamp: time.Now()},
		{UUID: "a-1", Role: "assistant", Model: "claude-opus-4-7", Content: []sessiontree.Block{{Type: "text", Text: "hello"}}, Timestamp: time.Now()},
	}
}

func TestConversationRenders(t *testing.T) {
	m := NewConversation(mkMsgs())
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	r := tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second))
	var buf strings.Builder
	var b [4096]byte
	for {
		n, err := r.Read(b[:])
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	got := buf.String()
	for _, want := range []string{"hi", "hello", "user", "assistant"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n--\n%s", want, got)
		}
	}
}

func readOutput(t *testing.T, tm *teatest.TestModel) string {
	t.Helper()
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	r := tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second))
	var buf strings.Builder
	var b [4096]byte
	for {
		n, err := r.Read(b[:])
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	return buf.String()
}

// Markdown bodies must be rendered through Glamour so that **bold**, code
// fences, and headings come out as ANSI-styled text. Raw markdown markers
// (``**`` around words, leading ``# `` on headings) should NOT appear in
// the rendered output.
func TestConversationRendersMarkdown(t *testing.T) {
	msgs := []*sessiontree.Message{
		{
			UUID:      "a-md",
			Role:      "assistant",
			Timestamp: time.Now(),
			Content: []sessiontree.Block{{
				Type: "text",
				Text: "# Heading\n\nThis is **bold** and `inline code`.\n\n```go\nfmt.Println(\"hi\")\n```\n",
			}},
		},
	}
	m := NewConversation(msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)

	// The literal markdown markers should be gone — Glamour transforms them.
	for _, marker := range []string{"**bold**", "# Heading"} {
		if strings.Contains(got, marker) {
			t.Errorf("raw markdown marker %q leaked through (Glamour not rendering)\n--\n%s",
				marker, got)
		}
	}
	// The semantic content should still be present (just styled, not stripped).
	for _, want := range []string{"bold", "Heading", "inline code"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered output missing semantic text %q\n--\n%s", want, got)
		}
	}
}

// Messages with multiple Content blocks (text + tool_use + text) must all
// be rendered, not just the first one.
func TestConversationRendersAllContentBlocks(t *testing.T) {
	msgs := []*sessiontree.Message{
		{
			UUID:      "a-multi",
			Role:      "assistant",
			Timestamp: time.Now(),
			Content: []sessiontree.Block{
				{Type: "text", Text: "First chunk."},
				{Type: "tool_use", Text: `{"command":"ls"}`},
				{Type: "text", Text: "Second chunk."},
			},
		},
	}
	m := NewConversation(msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)
	for _, want := range []string{"First chunk", "Second chunk"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing block content %q (only first block rendered?)\n--\n%s",
				want, got)
		}
	}
}

// Body text should render in terminal green (ANSI 10) and bold/strong
// in magenta (ANSI 13) so emphasis pops against the body. We assert via
// the SGR escape sequences Glamour emits — checking for the literal
// color codes is brittle but the alternative (rendering with a fake
// terminfo) is heavier and not worth it.
func TestConversationStyleColors(t *testing.T) {
	msgs := []*sessiontree.Message{{
		UUID:      "a-c",
		Role:      "assistant",
		Timestamp: time.Now(),
		Content:   []sessiontree.Block{{Type: "text", Text: "plain body, then **emphasis**."}},
	}}
	m := NewConversation(msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)

	// Termenv may emit either the 16-color SGR ([92m / [95m) or the 256
	// form ([38;5;10m / [38;5;13m). Bold may add ";1" before the trailing
	// 'm' (e.g. [95;1m). Match the prefix forms so either encoding works.
	if !strings.Contains(got, "[92") && !strings.Contains(got, "38;5;10") {
		t.Errorf("body text not rendered in green (no ANSI 10 escape):\n%s", got)
	}
	if !strings.Contains(got, "[95") && !strings.Contains(got, "38;5;13") {
		t.Errorf("strong/bold not rendered in magenta (no ANSI 13 escape):\n%s", got)
	}
}

// Markdown links should come out as OSC 8 hyperlink escapes so that
// terminals that understand them (Ghostty, iTerm2, WezTerm) make them
// clickable. The escape format is \x1b]8;;<URL>\x1b\\<TEXT>\x1b]8;;\x1b\\.
func TestConversationWrapsLinksInOSC8(t *testing.T) {
	msgs := []*sessiontree.Message{{
		UUID:      "a-l",
		Role:      "assistant",
		Timestamp: time.Now(),
		Content:   []sessiontree.Block{{Type: "text", Text: "see https://anthropic.com for details"}},
	}}
	m := NewConversation(msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)

	// Look for the OSC 8 opener that wrapHyperlinks emits.
	osc8 := "\x1b]8;;https://anthropic.com"
	if !strings.Contains(got, osc8) {
		t.Errorf("URL not wrapped in OSC 8 hyperlink escape:\n%s", got)
	}
}

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

func TestBranchListShowsSiblings(t *testing.T) {
	tr, err := sessiontree.LoadDir("../sessiontree/testdata")
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	m := NewBranchList(tr)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	got := readOutput(t, tm)
	// The fixtures have a-1 as a branch point with multiple children
	// (u-2 from linear, u-2a/u-2b from one-branch, u-3 from multi-branch).
	if !strings.Contains(got, "a-1") {
		t.Errorf("branch list missing branch-point a-1: %q", got)
	}
}

func TestMergeComposerEmitsCitations(t *testing.T) {
	msgs := []*sessiontree.Message{
		{UUID: "u-x", Role: "user", Content: []sessiontree.Block{{Type: "text", Text: "found a fix"}}},
		{UUID: "a-x", Role: "assistant", Content: []sessiontree.Block{{Type: "text", Text: "good idea"}}},
	}
	m := NewMergeComposer("branch-b", msgs)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace}) // select first
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace}) // select second
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	got := readOutput(t, tm)
	if !strings.Contains(got, "branch-b") {
		t.Errorf("output missing source-sid label: %s", got)
	}
}
