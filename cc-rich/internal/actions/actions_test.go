package actions

import (
	"os"
	"strings"
	"testing"
	"time"
)

type mockRunner struct {
	calls [][]string
}

func (m *mockRunner) Cmd(name string, args ...string) error {
	m.calls = append(m.calls, append([]string{name}, args...))
	return nil
}

// TestForkResumeShellsReplayShim — F1 dispatches via cc-replay-shim with a
// synthesized seed file (preamble + context). The seed file must exist on
// disk after Fork returns.
func TestForkResumeShellsReplayShim(t *testing.T) {
	dir, _ := os.MkdirTemp("", "cc-fork-")
	defer os.RemoveAll(dir)
	seed := dir + "/seed.txt"

	mr := &mockRunner{}
	err := Fork(mr, ForkResume, ForkArgs{
		SessionName:  "main",
		OrigCwd:      "/tmp/repo",
		SeedPath:     seed,
		PreambleText: "// continuing from session abc:msg-u-3\n\nlast turn: hello\n",
	})
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if len(mr.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(mr.calls))
	}
	got := strings.Join(mr.calls[0], " ")
	if !strings.Contains(got, "tmux new-window") {
		t.Errorf("call = %q, want tmux new-window", got)
	}
	if !strings.Contains(got, "cc-replay-shim "+seed) {
		t.Errorf("call = %q, want cc-replay-shim %s", got, seed)
	}
	body, err := os.ReadFile(seed)
	if err != nil {
		t.Fatalf("seed file not written: %v", err)
	}
	if !strings.Contains(string(body), "continuing from session abc") {
		t.Errorf("seed body missing preamble: %q", string(body))
	}
}

// TestForkReplayShellsReplayShim — F2 dispatches via cc-replay-shim with
// a pre-existing seed file. Fork does NOT write the file (caller's job).
func TestForkReplayShellsReplayShim(t *testing.T) {
	mr := &mockRunner{}
	if err := Fork(mr, ForkReplay, ForkArgs{
		SessionName: "main",
		OrigCwd:     "/tmp/repo",
		SeedPath:    "/tmp/seed.txt",
	}); err != nil {
		t.Fatalf("Fork: %v", err)
	}
	got := strings.Join(mr.calls[0], " ")
	if !strings.Contains(got, "cc-replay-shim /tmp/seed.txt") {
		t.Errorf("call = %q, want cc-replay-shim /tmp/seed.txt", got)
	}
}

// TestForkWorktreeRunsTwoCommands — F3 = F1 + git worktree add prefix.
func TestForkWorktreeRunsTwoCommands(t *testing.T) {
	dir, _ := os.MkdirTemp("", "cc-fork-")
	defer os.RemoveAll(dir)
	seed := dir + "/seed.txt"

	mr := &mockRunner{}
	err := Fork(mr, ForkWorktree, ForkArgs{
		SessionName:  "main",
		OrigCwd:      "/tmp/repo",
		SeedPath:     seed,
		PreambleText: "// continuing from session abc:msg-u-3\n",
		Branch:       "feat/x",
		WorktreeDir:  "/tmp/repo/.worktrees/feat-x",
	})
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if len(mr.calls) != 2 {
		t.Fatalf("calls = %d, want 2 (worktree add + tmux new-window)", len(mr.calls))
	}
	if mr.calls[0][0] != "git" || mr.calls[0][1] != "worktree" || mr.calls[0][2] != "add" {
		t.Errorf("first call = %v, want git worktree add ...", mr.calls[0])
	}
	got := strings.Join(mr.calls[1], " ")
	if !strings.Contains(got, "cc-replay-shim "+seed) {
		t.Errorf("second call = %q, want cc-replay-shim %s", got, seed)
	}
	if !strings.Contains(got, "-c /tmp/repo/.worktrees/feat-x") {
		t.Errorf("second call cwd should be the worktree dir; got %q", got)
	}
}

// TestForkValidates — every mode rejects empty required fields.
func TestForkValidates(t *testing.T) {
	mr := &mockRunner{}
	cases := []struct {
		name string
		mode ForkMode
		args ForkArgs
	}{
		{"resume-no-seed", ForkResume, ForkArgs{SessionName: "m", OrigCwd: "/", PreambleText: "x"}},
		{"resume-no-preamble", ForkResume, ForkArgs{SessionName: "m", OrigCwd: "/", SeedPath: "/tmp/s"}},
		{"replay-no-seed", ForkReplay, ForkArgs{SessionName: "m", OrigCwd: "/"}},
		{"worktree-no-branch", ForkWorktree, ForkArgs{SessionName: "m", OrigCwd: "/", SeedPath: "/tmp/s", PreambleText: "x", WorktreeDir: "/tmp/wt"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := Fork(mr, c.mode, c.args); err == nil {
				t.Errorf("%s: expected validation error, got nil", c.name)
			}
		})
	}
}

func TestQuoteToBufferAppends(t *testing.T) {
	dir, _ := os.MkdirTemp("", "ccq-")
	defer os.RemoveAll(dir)
	buf := dir + "/buffer.md"

	first := QuoteEntry{
		SessionID: "abc",
		MsgUUID:   "u-1",
		Timestamp: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		Text:      "hello",
	}
	if err := QuoteToBuffer(buf, first); err != nil {
		t.Fatalf("QuoteToBuffer: %v", err)
	}
	second := QuoteEntry{
		SessionID: "abc",
		MsgUUID:   "u-2",
		Timestamp: time.Date(2026, 5, 1, 12, 5, 0, 0, time.UTC),
		Text:      "follow up",
	}
	if err := QuoteToBuffer(buf, second); err != nil {
		t.Fatalf("QuoteToBuffer: %v", err)
	}
	body, err := os.ReadFile(buf)
	if err != nil {
		t.Fatalf("read buffer: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "hello") || !strings.Contains(got, "follow up") {
		t.Errorf("buffer missing both entries: %q", got)
	}
	if !strings.Contains(got, "u-1") || !strings.Contains(got, "u-2") {
		t.Errorf("buffer missing UUIDs: %q", got)
	}
}

func TestWriteMergeBufferFormat(t *testing.T) {
	dir, _ := os.MkdirTemp("", "ccm-")
	defer os.RemoveAll(dir)
	target := dir + "/.cc-pending-prompt"

	citations := []Citation{
		{
			SourceSID: "branch-b",
			MsgUUID:   "msg-x",
			Text:      "discovered: foo is broken because bar",
		},
		{
			SourceSID: "branch-b",
			MsgUUID:   "msg-y",
			Text:      "fix: use baz instead of bar",
		},
	}
	if err := WriteMergeBuffer(target, citations); err != nil {
		t.Fatalf("WriteMergeBuffer: %v", err)
	}
	body, _ := os.ReadFile(target)
	got := string(body)
	for _, want := range []string{
		"branch-b", "msg-x", "msg-y",
		"discovered: foo", "fix: use baz",
		"> ", // blockquote prefix
	} {
		if !strings.Contains(got, want) {
			t.Errorf("merge buffer missing %q: %q", want, got)
		}
	}
}

func TestWriteMergeBufferRejectsEmpty(t *testing.T) {
	dir, _ := os.MkdirTemp("", "ccm-")
	defer os.RemoveAll(dir)
	if err := WriteMergeBuffer(dir+"/x", nil); err == nil {
		t.Error("expected error on empty citations, got nil")
	}
}
