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
	// Drop the literal "# ", "## ", … prefixes too. Dark theme keeps
	// them as a visual cue, but in the sidebar the color + bold
	// already mark the heading; the leading hashes just look like
	// unrendered markdown to the reader.
	cfg.H1.Prefix = ""
	cfg.H1.Suffix = ""
	cfg.H2.Prefix = ""
	cfg.H2.Suffix = ""
	cfg.H3.Prefix = ""
	cfg.H3.Suffix = ""
	cfg.H4.Prefix = ""
	cfg.H4.Suffix = ""
	cfg.H5.Prefix = ""
	cfg.H5.Suffix = ""
	cfg.H6.Prefix = ""
	cfg.H6.Suffix = ""
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

	// rowCache memoizes per-message rendered rows keyed by msg UUID.
	// Cleared when width changes (markdown wrap is width-dependent).
	rowCache map[string]string
	// msgLineStart[i] is the line number where m.msgs[i]'s rendered
	// row begins in the assembled content. Computed once in
	// buildContent. j/k cursor navigation uses this for O(1)
	// scroll-to-message instead of re-rendering content.
	msgLineStart []int
	// contentBuilt is true after buildContent has run at least once.
	// We avoid rebuilding on cursor moves — content doesn't change,
	// only the viewport offset does.
	contentBuilt bool
}

func NewConversation(msgs []*sessiontree.Message) ConversationModel {
	return ConversationModel{msgs: msgs}
}

func (m ConversationModel) Init() tea.Cmd { return nil }

func (m ConversationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch t := msg.(type) {
	case tea.WindowSizeMsg:
		// Width change invalidates per-message render cache + line
		// offsets (Glamour wrap and lipgloss border are width-tuned).
		// Height change does not.
		widthChanged := t.Width != m.width
		if widthChanged {
			m.rowCache = nil
			m.contentBuilt = false
		}
		m.width = t.Width
		m.height = t.Height
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
		if !m.contentBuilt {
			m.vp.SetContent(m.buildContent())
			m.contentBuilt = true
		}

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
			// Advance cursor + scroll viewport to it. No content
			// rebuild — just an offset change.
			if m.cursor < len(m.msgs)-1 {
				m.cursor++
				if m.ready && m.cursor < len(m.msgLineStart) {
					m.vp.SetYOffset(m.msgLineStart[m.cursor])
				}
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				if m.ready && m.cursor < len(m.msgLineStart) {
					m.vp.SetYOffset(m.msgLineStart[m.cursor])
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

// buildContent renders all messages into a single string and
// cursor row decorated by a magenta rounded border. Returned string is
// what gets fed to the viewport in subsequent versions of this model.
//
// Pulled out of View() so that the viewport-aware version of this model
// can call buildContent() on cursor / width / msg-list changes and pass
// the result to viewport.SetContent — instead of rebuilding from inside
// View() (which Bubble Tea calls every frame, where work is wasteful).
// renderRow renders header + body + buttons for one msg WITHOUT the
// cursor border. The result is safe to cache because nothing in it
// depends on m.cursor — only on the message and current width.
func (m *ConversationModel) renderRow(msg *sessiontree.Message) string {
	header := fmt.Sprintf("%-9s  %s", msg.Role, msg.UUID)
	if msg.Role == "assistant" {
		header = StyleAsst.Render(header)
	} else {
		header = StyleUser.Render(header)
	}

	var bodyParts []string
	for _, b := range msg.Content {
		switch b.Type {
		case "text":
			bodyParts = append(bodyParts, m.renderMarkdown(b.Text))
		case "thinking":
			bodyParts = append(bodyParts, StyleMuted.Render("· thinking ·\n"+m.renderMarkdown(b.Text)))
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
	return lipgloss.JoinVertical(lipgloss.Left, header, body, buttons)
}

func (m *ConversationModel) buildContent() string {
	if len(m.msgs) == 0 {
		return StyleMuted.Render("(empty session)")
	}
	if m.rowCache == nil {
		m.rowCache = make(map[string]string, len(m.msgs))
	}
	m.msgLineStart = make([]int, len(m.msgs))
	var sb strings.Builder
	line := 0
	for i, msg := range m.msgs {
		m.msgLineStart[i] = line
		row, ok := m.rowCache[msg.UUID]
		if !ok {
			row = m.renderRow(msg)
			m.rowCache[msg.UUID] = row
		}
		sb.WriteString(row)
		sb.WriteString("\n\n")
		line += strings.Count(row, "\n") + 2
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
	// Reserve the bottom row for the footer; viewport already sized
	// itself to Height-1 in Update, so we just stack the viewport +
	// footer with lipgloss.
	return lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), m.footer())
}
