// cc-rich/internal/actions/fork.go
package actions

import (
	"fmt"
	"os"
)

// ForkMode selects which fork mechanic to use.
type ForkMode int

const (
	// ForkResume: F1 — "resume + branch" semantics. Writes a seed file
	// containing PreambleText, then opens a fresh `claude` in a new tmux
	// window via cc-replay-shim. Carries forward synthesized context.
	ForkResume ForkMode = iota

	// ForkReplay: F2 — "replay as prompt" semantics. Caller supplies an
	// already-written SeedPath containing the bare message text. New
	// session has zero prior context.
	ForkReplay

	// ForkWorktree: F3 — F1 + `git worktree add` first.
	ForkWorktree
)

// ForkArgs collects everything needed to dispatch a fork.
type ForkArgs struct {
	SessionName  string // tmux session name (target for new-window)
	OrigCwd      string // -c flag for new-window (or worktree dir for F3)
	SeedPath     string // path to seed file passed to cc-replay-shim
	PreambleText string // for ForkResume / ForkWorktree: written to SeedPath
	Branch       string // for ForkWorktree: branch to check out
	WorktreeDir  string // for ForkWorktree: filesystem path for `git worktree add`
}

// Fork dispatches one of F1/F2/F3 through the given Runner. For F1 and F3,
// it writes the seed file atomically before the dispatch. For F2, the
// caller is responsible for writing SeedPath.
func Fork(r Runner, mode ForkMode, args ForkArgs) error {
	switch mode {
	case ForkResume:
		if args.SeedPath == "" || args.PreambleText == "" {
			return fmt.Errorf("Fork(Resume): SeedPath and PreambleText required")
		}
		if err := writeSeedAtomic(args.SeedPath, args.PreambleText); err != nil {
			return err
		}
		shell := fmt.Sprintf("cc-replay-shim %s", args.SeedPath)
		return r.Cmd("tmux", "new-window", "-t", args.SessionName, "-c", args.OrigCwd, shell)

	case ForkReplay:
		if args.SeedPath == "" {
			return fmt.Errorf("Fork(Replay): SeedPath required")
		}
		shell := fmt.Sprintf("cc-replay-shim %s", args.SeedPath)
		return r.Cmd("tmux", "new-window", "-t", args.SessionName, "-c", args.OrigCwd, shell)

	case ForkWorktree:
		if args.Branch == "" || args.WorktreeDir == "" || args.SeedPath == "" || args.PreambleText == "" {
			return fmt.Errorf("Fork(Worktree): Branch, WorktreeDir, SeedPath, PreambleText required")
		}
		if err := r.Cmd("git", "worktree", "add", args.WorktreeDir, args.Branch); err != nil {
			return err
		}
		if err := writeSeedAtomic(args.SeedPath, args.PreambleText); err != nil {
			return err
		}
		shell := fmt.Sprintf("cc-replay-shim %s", args.SeedPath)
		return r.Cmd("tmux", "new-window", "-t", args.SessionName, "-c", args.WorktreeDir, shell)
	}
	return fmt.Errorf("unknown ForkMode %d", mode)
}

// writeSeedAtomic writes body to path via temp + rename so a partial
// write can never be observed by cc-replay-shim.
func writeSeedAtomic(path, body string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
