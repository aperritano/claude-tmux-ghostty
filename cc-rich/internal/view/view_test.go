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
