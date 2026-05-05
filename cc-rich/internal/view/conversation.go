// cc-rich/internal/view/conversation.go
package view

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"

	"github.com/aperritano/cc-rich/internal/actions"
	"github.com/aperritano/cc-rich/internal/sessiontree"
)

// urlRegex matches plain http(s) URLs in already-rendered text. We use
// it to wrap them in OSC 8 hyperlink escapes after Glamour rendering.
// The character class is permissive — common URL chars only, stops at
// whitespace/closing-paren/closing-bracket so we don't capture trailing
// punctuation in markdown-rendered prose.
var urlRegex = regexp.MustCompile(`https?://[^\s)\]]+`)

// prRefRegex matches GitHub-style references like "PR-828", "issue-429",
// "pr_42". Requires an explicit separator so plain digits like "PR828"
// don't accidentally match. Word boundaries on both ends keep us out
// of substrings ("PR-828abc" stays unmatched). Skips bare "#N" — too
// ambiguous (HTML anchors, hex colors) without surrounding-context
// detection that Go's regexp can't easily express.
var prRefRegex = regexp.MustCompile(`(?i)\b(PR|issue)[-_](\d+)\b`)

// osc8Wrap returns the OSC 8 hyperlink escape wrapping `text` with
// `url`. Terminals that don't understand OSC 8 see only the text.
//
//	format: \x1b]8;;<URL>\x1b\\<TEXT>\x1b]8;;\x1b\\
func osc8Wrap(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// wrapHyperlinks rewrites plain URLs in s as OSC 8 hyperlink escapes so
// terminals that understand them (Ghostty, iTerm2, WezTerm, modern
// Konsole) make them clickable. tmux passes OSC 8 through with default
// config in recent versions.
func wrapHyperlinks(s string) string {
	return urlRegex.ReplaceAllStringFunc(s, func(u string) string {
		return osc8Wrap(u, u)
	})
}

// parseGithubSlug extracts "owner/repo" from a GitHub remote URL.
// Returns "" for non-GitHub remotes or unparseable input.
func parseGithubSlug(remote string) string {
	var s string
	switch {
	case strings.HasPrefix(remote, "https://github.com/"):
		s = strings.TrimPrefix(remote, "https://github.com/")
	case strings.HasPrefix(remote, "git@github.com:"):
		s = strings.TrimPrefix(remote, "git@github.com:")
	default:
		return ""
	}
	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimSuffix(s, "/")
	return s
}

// repoSlug returns "owner/repo" for cwd's git origin (cached per cwd).
// Returns "" if cwd is empty, isn't a git repo, or its origin isn't
// GitHub. Caching matters because a typical session has thousands of
// messages from a small handful of cwds.
func (m *ConversationModel) repoSlug(cwd string) string {
	if cwd == "" {
		return ""
	}
	if m.repoSlugCache == nil {
		m.repoSlugCache = map[string]string{}
	}
	if slug, ok := m.repoSlugCache[cwd]; ok {
		return slug
	}
	out, err := exec.Command("git", "-C", cwd, "remote", "get-url", "origin").Output()
	if err != nil {
		m.repoSlugCache[cwd] = ""
		return ""
	}
	slug := parseGithubSlug(strings.TrimSpace(string(out)))
	m.repoSlugCache[cwd] = slug
	return slug
}

// wrapPRRefs links GitHub-style PR/issue references (PR-N, issue-N) to
// the matching pull/issue page. Requires a repo slug from the message's
// cwd; refs in messages without a GitHub remote are left as plain text.
func (m *ConversationModel) wrapPRRefs(s, cwd string) string {
	slug := m.repoSlug(cwd)
	if slug == "" {
		return s
	}
	return prRefRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := prRefRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		kind := strings.ToLower(parts[1])
		num := parts[2]
		path := "pull"
		if kind == "issue" {
			path = "issues"
		}
		url := fmt.Sprintf("https://github.com/%s/%s/%s", slug, path, num)
		return osc8Wrap(url, match)
	})
}

// filePathRegex matches relative-or-absolute file path tokens with at
// least one slash and a 1-8 char extension. Skips bare filenames
// (e.g., "Makefile", "README") because matching every word risks too
// many false positives. Existence-check in wrapFilePaths filters the
// rest.
var filePathRegex = regexp.MustCompile(`[\w./~-]+/[\w./-]*[\w]+\.[a-zA-Z][a-zA-Z0-9]{0,8}(?:[:](\d+))?\b`)

// resolvePath turns a possibly-relative path into an absolute path
// using cwd as the anchor and expanding ~/. Returns "" if the input
// can't reasonably be turned into a path (no cwd, etc.).
func resolvePath(p, cwd string) string {
	if strings.HasPrefix(p, "~/") {
		if home := os.Getenv("HOME"); home != "" {
			return filepath.Join(home, p[2:])
		}
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	if cwd == "" {
		return ""
	}
	return filepath.Join(cwd, p)
}

// wrapFilePaths links file path tokens to vscode://file/<abs-path> so
// clicking opens them in VS Code. The path must (a) match the regex,
// (b) resolve to a path that exists on disk relative to msg.cwd. False
// positives (e.g. "github.com/foo/bar.go" inside an already-wrapped
// URL OSC 8) self-eliminate via the existence check.
//
// Captures :N line-number suffix when present; passes through to
// vscode://file/abs:N which jumps to that line.
func (m *ConversationModel) wrapFilePaths(s, cwd string) string {
	if cwd == "" {
		return s
	}
	return filePathRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Strip optional :N suffix for path resolution; preserve it
		// in the URL so VS Code jumps to the line.
		path := match
		lineSuffix := ""
		if idx := strings.LastIndex(match, ":"); idx > 0 {
			tail := match[idx+1:]
			if _, err := fmt.Sscanf(tail, "%d", new(int)); err == nil {
				path = match[:idx]
				lineSuffix = match[idx:]
			}
		}
		abs := resolvePath(path, cwd)
		if abs == "" {
			return match
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			return match
		}
		url := "vscode://file" + abs + lineSuffix
		return osc8Wrap(url, match)
	})
}

// linkify wraps every link-like token in s with OSC 8 escapes so the
// terminal makes them clickable. Order matters: full URLs first (most
// specific), then PR/issue refs (cwd-dependent), then local file
// paths (existence-checked, opens in VS Code).
func (m *ConversationModel) linkify(s, cwd string) string {
	s = wrapHyperlinks(s)
	s = m.wrapPRRefs(s, cwd)
	s = m.wrapFilePaths(s, cwd)
	return s
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

// cmenuState holds the state of the right-click context menu overlay.
// visible=false means the menu is not shown.
type cmenuState struct {
	visible bool
	sel     int // 0=resume+branch 1=replay 2=quote 3=merge
	msgIdx  int // which message this menu targets
	x, y    int // screen position of the right-click (0-indexed)
}

var cmenuLabels = [4]string{
	"[1] Resume + branch",
	"[2] Replay as prompt",
	"[4] Quote to buffer",
	"[m] Merge composer",
}

// PendingAction carries a fork request to dispatch after the TUI exits.
// main.go reads this from the final model after p.Run() returns and
// executes via actions.Fork with a DefaultRunner.
type PendingAction struct {
	Kind      string // "fork_resume" | "fork_replay"
	Msg       *sessiontree.Message
	SessionID string
}

// ConversationModel renders a list of messages as a scrollable column.
// The actual scroll state (YOffset) lives in viewport; we own cursor
// position and the message list and feed the viewport pre-rendered
// content via SetContent on state changes.
type ConversationModel struct {
	msgs      []*sessiontree.Message
	cursor    int
	width     int
	height    int
	done      bool
	sessionID string // Claude session UUID — carried to PendingAction for fork
	md        *glamour.TermRenderer // built lazily on first WindowSizeMsg
	mdW       int                   // width the renderer was built for

	vp    viewport.Model // owns scroll position; we feed it content
	ready bool           // becomes true after first WindowSizeMsg

	// rowCache memoizes per-message rendered rows keyed by msg UUID.
	// Cleared when width changes (markdown wrap is width-dependent).
	rowCache map[string]string
	// repoSlugCache memoizes "owner/repo" per cwd. A session has
	// thousands of messages from a small set of cwds; spawning git for
	// each render would dominate cold-start time.
	repoSlugCache map[string]string
	// msgLineStart[i] is the line number where m.msgs[i]'s rendered
	// row begins in the assembled content. Computed once in
	// buildContent. j/k cursor navigation uses this for O(1)
	// scroll-to-message instead of re-rendering content.
	msgLineStart []int
	// contentBuilt is true after buildContent has run at least once.
	// We avoid rebuilding on cursor moves — content doesn't change,
	// only the viewport offset does.
	contentBuilt bool

	// Context menu overlay state.
	cmenu cmenuState
	// toastMsg shows a brief notice in the footer for one frame (cleared
	// after being rendered once).
	toastMsg string

	// PendingAction is set before tea.Quit for fork actions (F1/F2).
	// main.go reads this after p.Run() and dispatches via actions.Fork.
	PendingAction *PendingAction
}

func NewConversation(sessionID string, msgs []*sessiontree.Message) ConversationModel {
	return ConversationModel{sessionID: sessionID, msgs: msgs}
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

	case tea.MouseMsg:
		if t.Button == tea.MouseButtonRight && t.Action == tea.MouseActionPress {
			// Map screen Y → content line → message index.
			contentLine := m.vp.YOffset + t.Y
			msgIdx := m.msgIdxAtContentLine(contentLine)
			m.cmenu = cmenuState{
				visible: true,
				sel:     0,
				msgIdx:  msgIdx,
				x:       t.X,
				y:       t.Y,
			}
			return m, nil
		}

	case tea.KeyMsg:
		if m.cmenu.visible {
			// Context menu key handling.
			switch t.String() {
			case "esc", "q":
				m.cmenu.visible = false
				return m, nil
			case "j", "down":
				m.cmenu.sel = (m.cmenu.sel + 1) % 4
				return m, nil
			case "k", "up":
				m.cmenu.sel = (m.cmenu.sel + 3) % 4
				return m, nil
			case "enter":
				return m.executeAction(m.cmenu.sel)
			case "1":
				return m.executeAction(0)
			case "2":
				return m.executeAction(1)
			case "4":
				return m.executeAction(2)
			case "m":
				return m.executeAction(3)
			}
			return m, nil
		}

		// Normal key handling when menu is not visible.
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
		case "1":
			// F1: open menu pre-selecting "Resume + branch"
			m.cmenu = cmenuState{visible: true, sel: 0, msgIdx: m.cursor, x: 2, y: 2}
			return m, nil
		case "2":
			// F2: open menu pre-selecting "Replay as prompt"
			m.cmenu = cmenuState{visible: true, sel: 1, msgIdx: m.cursor, x: 2, y: 2}
			return m, nil
		case "4":
			// F4: quote immediately — no menu needed for this single-step action
			if m.cursor < len(m.msgs) {
				if err := m.doQuote(m.msgs[m.cursor]); err != nil {
					m.toastMsg = "quote failed: " + err.Error()
				} else {
					m.toastMsg = "quoted → ~/.local/share/cc-rich/quotes.md"
				}
			}
			return m, nil
		case "m":
			m.toastMsg = "merge composer not yet implemented"
			return m, nil
		}
	}

	// Forward unhandled keys + mouse to viewport so PgUp/PgDn/wheel
	// work without us having to enumerate them. Context menu intercepts
	// all keys when visible (returns early above), so only non-menu
	// events reach here.
	if m.ready && !m.cmenu.visible {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// msgIdxAtContentLine returns the index of the message whose rendered
// block begins at or before contentLine. Used to map right-click Y
// coordinate (+ viewport offset) to a message.
func (m *ConversationModel) msgIdxAtContentLine(contentLine int) int {
	if len(m.msgLineStart) == 0 {
		return 0
	}
	idx := 0
	for i, start := range m.msgLineStart {
		if start <= contentLine {
			idx = i
		} else {
			break
		}
	}
	return idx
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
	return strings.TrimRight(out, "\n")
}

// footerLine renders a one-line scroll-position indicator, or a brief
// toast message for one render cycle.
func (m ConversationModel) footerLine() string {
	if m.toastMsg != "" {
		return StyleMuted.Render("✓ " + m.toastMsg)
	}
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

// renderContextMenu returns a lipgloss-bordered context menu string
// with the currently selected item highlighted.
func (m ConversationModel) renderContextMenu() string {
	menuBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent).
		Padding(0, 1)
	selStyle := lipgloss.NewStyle().
		Background(ColorAccent).
		Foreground(lipgloss.Color("0")).
		Bold(true)

	var lines []string
	msg := ""
	if m.cmenu.msgIdx < len(m.msgs) {
		msg = m.msgs[m.cmenu.msgIdx].UUID
		if len(msg) > 12 {
			msg = msg[:12]
		}
	}
	lines = append(lines, StyleMuted.Render("msg: "+msg))
	for i, label := range cmenuLabels {
		if i == m.cmenu.sel {
			lines = append(lines, selStyle.Render("  "+label+"  "))
		} else {
			lines = append(lines, "  "+label)
		}
	}
	lines = append(lines, StyleMuted.Render("↑↓ navigate · enter/1/2/4/m · esc cancel"))
	return menuBorder.Render(strings.Join(lines, "\n"))
}

// overlayContextMenu embeds the context menu over base using ANSI cursor
// positioning, clamped to the screen bounds. AltScreen mode means the
// framework clears the screen each frame so there are no artifacts.
func (m ConversationModel) overlayContextMenu(base string) string {
	menu := m.renderContextMenu()
	menuLines := strings.Split(menu, "\n")
	menuW := 28 // approximate visible width of the menu box
	menuH := len(menuLines)

	startX := m.cmenu.x
	startY := m.cmenu.y
	if startX+menuW > m.width {
		startX = m.width - menuW
	}
	if startX < 0 {
		startX = 0
	}
	if startY+menuH > m.height-1 {
		startY = m.height - 1 - menuH
	}
	if startY < 0 {
		startY = 0
	}

	var sb strings.Builder
	sb.WriteString(base)
	for i, line := range menuLines {
		// \x1b[row;colH: ANSI cursor absolute positioning (1-indexed)
		sb.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", startY+i+1, startX+1, line))
	}
	return sb.String()
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
			bodyParts = append(bodyParts, m.linkify(m.renderMarkdown(b.Text), msg.Cwd))
		case "thinking":
			bodyParts = append(bodyParts, StyleMuted.Render("· thinking ·\n"+m.linkify(m.renderMarkdown(b.Text), msg.Cwd)))
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

// doQuote appends the message's text content to the shared quote buffer.
// Creates the directory and file if needed. Atomic: write to tmp + rename.
func (m *ConversationModel) doQuote(msg *sessiontree.Message) error {
	home := os.Getenv("HOME")
	bufDir := filepath.Join(home, ".local", "share", "cc-rich")
	if err := os.MkdirAll(bufDir, 0o755); err != nil {
		return err
	}
	bufPath := filepath.Join(bufDir, "quotes.md")

	var text strings.Builder
	for _, b := range msg.Content {
		if b.Type == "text" {
			text.WriteString(b.Text)
			text.WriteString("\n")
		}
	}
	return actions.QuoteToBuffer(bufPath, actions.QuoteEntry{
		SessionID: m.sessionID,
		MsgUUID:   msg.UUID,
		Timestamp: time.Now(),
		Text:      strings.TrimSpace(text.String()),
	})
}

// executeAction dispatches the selected context menu action.
// Fork actions (0, 1) set PendingAction and quit — main.go runs them.
// Quote (2) writes the buffer file inline and shows a toast.
// Merge (3) is not yet implemented.
func (m ConversationModel) executeAction(itemIdx int) (tea.Model, tea.Cmd) {
	m.cmenu.visible = false
	if m.cmenu.msgIdx >= len(m.msgs) {
		return m, nil
	}
	msg := m.msgs[m.cmenu.msgIdx]

	switch itemIdx {
	case 0: // Resume + branch (F1)
		m.PendingAction = &PendingAction{
			Kind:      "fork_resume",
			Msg:       msg,
			SessionID: m.sessionID,
		}
		m.done = true
		return m, tea.Quit

	case 1: // Replay as prompt (F2)
		m.PendingAction = &PendingAction{
			Kind:      "fork_replay",
			Msg:       msg,
			SessionID: m.sessionID,
		}
		m.done = true
		return m, tea.Quit

	case 2: // Quote to buffer (F4)
		if err := m.doQuote(msg); err != nil {
			m.toastMsg = "quote failed: " + err.Error()
		} else {
			m.toastMsg = "quoted → ~/.local/share/cc-rich/quotes.md"
		}

	case 3: // Merge composer
		m.toastMsg = "merge composer not yet implemented"
	}
	return m, nil
}

func (m ConversationModel) View() string {
	if !m.ready {
		// Pre-WindowSizeMsg: viewport has no size; fall back to the
		// raw content (terminal will clip it, but at least the user
		// sees something for the first frame).
		return m.buildContent()
	}
	// Clear toast after first render (shown for one frame).
	footer := m.footerLine()
	m.toastMsg = ""

	base := lipgloss.JoinVertical(lipgloss.Left, m.vp.View(), footer)
	if m.cmenu.visible {
		return m.overlayContextMenu(base)
	}
	return base
}
