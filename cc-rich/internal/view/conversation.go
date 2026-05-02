// cc-rich/internal/view/conversation.go
package view

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aperritano/cc-rich/internal/sessiontree"
)

// ConversationModel renders a list of messages as a scrollable column.
type ConversationModel struct {
	msgs   []*sessiontree.Message
	cursor int
	width  int
	height int
	done   bool
}

func NewConversation(msgs []*sessiontree.Message) ConversationModel {
	return ConversationModel{msgs: msgs}
}

func (m ConversationModel) Init() tea.Cmd { return nil }

func (m ConversationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = t.Width
		m.height = t.Height
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
		}
	}
	return m, nil
}

func (m ConversationModel) View() string {
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
		body := ""
		if len(msg.Content) > 0 {
			body = msg.Content[0].Text
		}
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
