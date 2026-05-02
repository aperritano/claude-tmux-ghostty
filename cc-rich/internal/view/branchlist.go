// cc-rich/internal/view/branchlist.go
package view

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aperritano/cc-rich/internal/sessiontree"
)

// BranchListModel shows the branch points (parents with multiple children)
// of a Tree. Selection is a UUID; the parent emits BranchSelected{UUID}
// to its container so the central pane can re-render that branch.
type BranchListModel struct {
	tr     *sessiontree.Tree
	bps    []string
	cursor int
	done   bool
}

// BranchSelected is emitted when the user picks a branch via Enter.
// The container (main TUI) listens and re-renders the conversation pane
// with the lineage rooted at the selected child UUID.
type BranchSelected struct{ UUID string }

func NewBranchList(tr *sessiontree.Tree) BranchListModel {
	return BranchListModel{tr: tr, bps: tr.BranchPoints()}
}

func (m BranchListModel) Init() tea.Cmd { return nil }

func (m BranchListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case tea.KeyMsg:
		switch t.String() {
		case "esc", "q":
			m.done = true
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.bps)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if len(m.bps) == 0 {
				return m, nil
			}
			parent := m.bps[m.cursor]
			kids := m.tr.Children[parent]
			if len(kids) > 0 {
				return m, func() tea.Msg { return BranchSelected{UUID: kids[0]} }
			}
		}
	}
	return m, nil
}

func (m BranchListModel) View() string {
	if len(m.bps) == 0 {
		return StyleMuted.Render("(no branches)")
	}
	var sb strings.Builder
	sb.WriteString(StyleHeader.Render("Branches") + "\n\n")
	for i, parent := range m.bps {
		kids := m.tr.Children[parent]
		marker := "  "
		if i == m.cursor {
			marker = "▸ "
		}
		fmt.Fprintf(&sb, "%s%s  ", marker, parent)
		for j, k := range kids {
			if j > 0 {
				sb.WriteString(" / ")
			}
			sb.WriteString(k)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
