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

	"github.com/aperritano/cc-rich/internal/actions"
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
	// Show every message in the transcript, ordered chronologically.
	// AllByTime (vs Lineage) includes parallel sub-agent branches —
	// in multi-agent sessions a 4000-message transcript can have
	// only ~50 messages on the latest turn's lineage. The user wants
	// to scroll the whole thing.
	msgs := tr.AllByTime()
	if len(msgs) == 0 {
		fmt.Println("(empty session)")
		return
	}
	p := tea.NewProgram(
		view.NewConversation(sid, msgs),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	final, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Dispatch any pending fork action (F1 / F2) set by the conversation
	// view before it called tea.Quit. Fork requires tmux subprocess calls
	// which must happen outside the Bubble Tea render loop.
	if conv, ok := final.(view.ConversationModel); ok && conv.PendingAction != nil {
		if err := dispatchFork(conv.PendingAction); err != nil {
			fmt.Fprintf(os.Stderr, "fork error: %v\n", err)
			os.Exit(1)
		}
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

// dispatchFork executes a pending fork action after the TUI has exited.
// fork_resume: synthesized preamble → seed file → cc-replay-shim in new tmux window.
// fork_replay: raw message text → seed file → cc-replay-shim in new tmux window.
// Both use atomic seed writes through actions.Fork so cc-replay-shim never
// sees a partial file.
func dispatchFork(a *view.PendingAction) error {
	tmpDir := os.TempDir()
	seedPath := filepath.Join(tmpDir, "cc-rich-seed-"+a.Msg.UUID[:8]+".txt")

	// Collect text content from the message.
	var textParts []string
	for _, b := range a.Msg.Content {
		if b.Type == "text" {
			textParts = append(textParts, b.Text)
		}
	}
	msgText := strings.Join(textParts, "\n")

	// Derive tmux session name (the fork opens a new window in the same session).
	sessOut, err := exec.Command("tmux", "display-message", "-p", "#{session_name}").Output()
	if err != nil {
		return fmt.Errorf("tmux session name: %w", err)
	}
	sessName := strings.TrimSpace(string(sessOut))
	cwd := a.Msg.Cwd
	if cwd == "" {
		cwd = os.Getenv("HOME")
	}

	r := actions.DefaultRunner{}
	switch a.Kind {
	case "fork_resume":
		preamble := "# Continuing from session " + a.SessionID + "\n\n" + msgText
		return actions.Fork(r, actions.ForkResume, actions.ForkArgs{
			SessionName:  sessName,
			OrigCwd:      cwd,
			SeedPath:     seedPath,
			PreambleText: preamble,
		})
	case "fork_replay":
		if err := os.WriteFile(seedPath, []byte(msgText), 0o644); err != nil {
			return err
		}
		return actions.Fork(r, actions.ForkReplay, actions.ForkArgs{
			SessionName: sessName,
			OrigCwd:     cwd,
			SeedPath:    seedPath,
		})
	}
	return fmt.Errorf("unknown fork kind: %s", a.Kind)
}
