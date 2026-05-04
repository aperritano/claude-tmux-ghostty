// cc-rich/internal/view/conversation.go
package view

import (
	"fmt"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"

	"github.com/aperritano/cc-rich/internal/sessiontree"
)

// urlRegex matches plain http(s) URLs in already-rendered text. We use
// it to wrap them in OSC 8 hyperlink escapes after Glamour rendering.
// The character class is permissive — common URL chars only, stops at
// whitespace/closing-paren/closing-bracket so we don't capture trailing
// punctuation in markdown-rendered prose.
var urlRegex = regexp.MustCompile(`https?://[^\s)\]]+`)

// wrapHyperlinks rewrites plain URLs in s as OSC 8 hyperlink escapes so
// terminals that understand them (Ghostty, iTerm2, WezTerm, modern
// Konsole) make them clickable. Terminals without OSC 8 just see the
// URL text unchanged. tmux passes OSC 8 through with default config in
// recent versions.
//
//	OSC 8 format: \x1b]8;;<URL>\x1b\\<TEXT>\x1b]8;;\x1b\\
func wrapHyperlinks(s string) string {
	return urlRegex.ReplaceAllStringFunc(s, func(u string) string {
		return "\x1b]8;;" + u + "\x1b\\" + u + "\x1b]8;;\x1b\\"
	})
}

// styleConfig builds the Glamour style: dark base, with body text set
// to terminal green and bold/strong set to the same magenta accent
// already used for active borders. Other categories inherit from dark.
func styleConfig() ansi.StyleConfig {
	cfg := styles.DarkStyleConfig
	green := "10"   // ANSI bright green — "terminal" body text
	magenta := "13" // ANSI bright magenta — matches ColorAccent
	yes := true

	cfg.Document.Color = &green
	cfg.Strong.Color = &magenta
	cfg.Strong.Bold = &yes
	// Drop the literal ** prefix/suffix the dark style adds around bold —
	// the color change carries the emphasis on its own.
	cfg.Strong.BlockPrefix = ""
	cfg.Strong.BlockSuffix = ""
	return cfg
}

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
		// Custom dark-derived style: green body, magenta bold. WithStyles
		// (vs WithStandardStyle) forces ANSI escapes regardless of TTY
		// detection, so test and production output match.
		r, err := glamour.NewTermRenderer(
			glamour.WithStyles(styleConfig()),
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
	return wrapHyperlinks(strings.TrimRight(out, "\n"))
}

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
