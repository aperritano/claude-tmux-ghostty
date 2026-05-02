// cc-rich/internal/view/conversation.go
package view

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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
	md     *glamour.TermRenderer // built lazily on first WindowSizeMsg
	mdW    int                   // width the renderer was built for
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

// renderMarkdown lazily builds a width-tuned Glamour renderer and runs the
// given markdown text through it. Falls back to the raw text if Glamour
// errors (so a malformed code fence can't blank the pane).
func (m *ConversationModel) renderMarkdown(md string) string {
	wrap := m.width - 4 // leave room for the row's rounded border + padding
	if wrap < 20 {
		wrap = 80 // pre-WindowSizeMsg or absurdly narrow pane
	}
	if m.md == nil || m.mdW != wrap {
		// "dark" forces ANSI styling regardless of TTY detection — tmux
		// always supports colors, and forcing the style makes test output
		// match production behavior. WithAutoStyle would fall back to
		// "notty" under non-TTY harnesses (teatest) and skip the styling.
		r, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(wrap),
		)
		if err != nil {
			return md
		}
		m.md = r
		m.mdW = wrap
	}
	out, err := m.md.Render(md)
	if err != nil {
		return md
	}
	return strings.TrimRight(out, "\n")
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
