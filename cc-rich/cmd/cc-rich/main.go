// Command cc-rich is a tmux-popup-overlay TUI for Claude Code sessions.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aperritano/cc-rich/internal/sessiontree"
	"github.com/aperritano/cc-rich/internal/view"
)

func main() {
	pane := flag.String("pane", "", "tmux pane id (e.g. %5) — resolve to active Claude session")
	browse := flag.Bool("browse", false, "list all known sessions instead of one")
	mergeInto := flag.String("merge-into", "", "open merge composer against the given pane's session")
	flag.Parse()

	switch {
	case *browse:
		fmt.Println("(browse mode — not yet implemented)")
		return
	case *mergeInto != "":
		fmt.Println("(merge mode — not yet implemented)")
		return
	case *pane != "":
		runPane(*pane)
	default:
		fmt.Fprintln(os.Stderr, "usage: cc-rich --pane %ID | --browse | --merge-into %ID")
		os.Exit(2)
	}
}

func runPane(paneID string) {
	sid, err := sessionFromPane(paneID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no Claude session in pane %s: %v\n", paneID, err)
		os.Exit(1)
	}
	path, err := transcriptPath(sid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "transcript for %s not found: %v\n", sid, err)
		os.Exit(1)
	}
	tr, err := sessiontree.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", path, err)
		os.Exit(1)
	}
	var last *sessiontree.Message
	for _, m := range tr.ByUUID {
		if last == nil || m.Timestamp.After(last.Timestamp) {
			last = m
		}
	}
	if last == nil {
		fmt.Println("(empty session)")
		return
	}
	msgs := tr.Lineage(last.UUID)
	p := tea.NewProgram(view.NewConversation(msgs), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// sessionFromPane shells out to ~/bin/tmux-claude-session and parses
// "claude:<sid>" from its output. Returns the bare sid.
// sessionFromPane shells out to `tmux-claude-session --bare <tty>` which
// emits the full session UUID with no decoration. (The helper's default
// mode emits a statusline-friendly "│ claude:<truncated>" string useful
// only for the bottom bar; --bare returns the full UUID we need for
// transcript lookup.)
func sessionFromPane(paneID string) (string, error) {
	tty, err := tmuxDisplay(paneID, "#{pane_tty}")
	if err != nil {
		return "", err
	}
	out, err := exec.Command(
		filepath.Join(os.Getenv("HOME"), "bin", "tmux-claude-session"),
		"--bare", tty,
	).Output()
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", fmt.Errorf("no Claude session on tty %s", tty)
	}
	return s, nil
}

func tmuxDisplay(target, format string) (string, error) {
	out, err := exec.Command("tmux", "display-message", "-t", target, "-p", format).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// transcriptPath finds <sid>.jsonl somewhere under ~/.claude/projects/.
func transcriptPath(sid string) (string, error) {
	root := filepath.Join(os.Getenv("HOME"), ".claude", "projects")
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Base(path) == sid+".jsonl" {
			found = path
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("not found")
	}
	return found, nil
}
