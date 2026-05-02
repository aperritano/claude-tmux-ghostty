package view

import (
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
