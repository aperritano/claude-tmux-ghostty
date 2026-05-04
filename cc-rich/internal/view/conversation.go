// cc-rich/internal/view/conversation.go
package view

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
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

	vp    viewport.Model // owns scroll position; we feed it content
	ready bool           // becomes true after first WindowSizeMsg
}

func NewConversation(msgs []*sessiontree.Message) ConversationModel {
	return ConversationModel{msgs: msgs}
}

func (m ConversationModel) Init() tea.Cmd { return nil }

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
	// Walk the full content one msg at a time; sum line counts to
	// find the cursor row's line number.
	full := m.buildContent()
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
	if !m.ready {
		// Pre-WindowSizeMsg: viewport has no size; fall back to the
		// raw content (terminal will clip it, but at least the user
		// sees something for the first frame).
		return m.buildContent()
	}
	return m.vp.View()
}
