// cc-rich/internal/view/merge.go
package view

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aperritano/cc-rich/internal/actions"
	"github.com/aperritano/cc-rich/internal/sessiontree"
)

// MergeComposerModel shows messages from a chosen sibling branch with
// checkboxes; pressing Enter emits a MergeCommit message containing the
// selected citations.
type MergeComposerModel struct {
	sourceSID string
	msgs      []*sessiontree.Message
	checked   map[int]bool
	cursor    int
	done      bool
}

// MergeCommit is emitted when the user presses Enter after selecting messages.
// The container is responsible for routing it to actions.WriteMergeBuffer.
type MergeCommit struct {
	Citations []actions.Citation
}

func NewMergeComposer(sourceSID string, msgs []*sessiontree.Message) MergeComposerModel {
	return MergeComposerModel{sourceSID: sourceSID, msgs: msgs, checked: make(map[int]bool)}
}

func (m MergeComposerModel) Init() tea.Cmd { return nil }

func (m MergeComposerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case tea.KeyMsg:
		switch t.String() {
		case "esc", "q":
			m.done = true
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.msgs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case " ", "space":
			m.checked[m.cursor] = !m.checked[m.cursor]
		case "enter":
			var cites []actions.Citation
			for i, on := range m.checked {
				if !on || i >= len(m.msgs) {
					continue
				}
				txt := ""
				if len(m.msgs[i].Content) > 0 {
					txt = m.msgs[i].Content[0].Text
				}
				cites = append(cites, actions.Citation{
					SourceSID: m.sourceSID,
					MsgUUID:   m.msgs[i].UUID,
					Text:      txt,
				})
			}
			m.done = true
			return m, tea.Sequence(func() tea.Msg { return MergeCommit{Citations: cites} }, tea.Quit)
		}
	}
	return m, nil
}

func (m MergeComposerModel) View() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s  source: %s\n\n", StyleHeader.Render("Merge composer"), m.sourceSID)
	for i, msg := range m.msgs {
		mark := "[ ]"
		if m.checked[i] {
			mark = "[x]"
		}
		cur := "  "
		if i == m.cursor {
			cur = "▸ "
		}
		body := ""
		if len(msg.Content) > 0 {
			body = msg.Content[0].Text
		}
		fmt.Fprintf(&sb, "%s%s %-9s  %s\n", cur, mark, msg.Role, body)
	}
	sb.WriteString("\n")
	sb.WriteString(StyleMuted.Render("[space] toggle  [enter] commit  [esc] cancel"))
	return sb.String()
}
