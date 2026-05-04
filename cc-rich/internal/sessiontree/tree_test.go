package sessiontree

import (
	"path/filepath"
	"testing"
)

func TestLoadLinear(t *testing.T) {
	tr, err := Load(filepath.Join("testdata", "linear.jsonl"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := len(tr.ByUUID), 3; got != want {
		t.Errorf("ByUUID len = %d, want %d", got, want)
	}
	for _, u := range []string{"u-1", "a-1", "u-2"} {
		if tr.ByUUID[u] == nil {
			t.Errorf("missing message %q", u)
		}
	}
	if got := tr.ByUUID["a-1"].ParentUUID; got != "u-1" {
		t.Errorf("a-1.parent = %q, want %q", got, "u-1")
	}
	if got := tr.ByUUID["a-1"].Role; got != "assistant" {
		t.Errorf("a-1.role = %q, want assistant", got)
	}
	if got := tr.ByUUID["a-1"].Model; got != "claude-opus-4-7" {
		t.Errorf("a-1.model = %q", got)
	}
	if got := tr.Children["u-1"]; len(got) != 1 || got[0] != "a-1" {
		t.Errorf("Children[u-1] = %v", got)
	}
	if got := tr.Roots; len(got) != 1 || got[0] != "u-1" {
		t.Errorf("Roots = %v, want [u-1]", got)
	}
}

func TestBranchPoints(t *testing.T) {
	tr, err := Load("testdata/one-branch.jsonl")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := tr.BranchPoints()
	if len(got) != 1 || got[0] != "a-1" {
		t.Errorf("BranchPoints = %v, want [a-1]", got)
	}
}

func TestLineage(t *testing.T) {
	tr, err := Load("testdata/one-branch.jsonl")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := tr.Lineage("u-2b")
	if len(got) != 3 {
		t.Fatalf("Lineage len = %d, want 3", len(got))
	}
	want := []string{"u-1", "a-1", "u-2b"}
	for i, m := range got {
		if m.UUID != want[i] {
			t.Errorf("Lineage[%d].UUID = %q, want %q", i, m.UUID, want[i])
		}
	}
}

func TestLoadTruncated(t *testing.T) {
	tr, err := Load("testdata/truncated.jsonl")
	if err != nil {
		t.Fatalf("Load returned error on truncated file: %v", err)
	}
	if got := len(tr.ByUUID); got != 2 {
		t.Errorf("ByUUID len = %d, want 2 (truncated last line skipped)", got)
	}
}

// Real Claude transcripts interleave chat messages with attachment /
// metadata records that have parentUuids forming part of the chain
// but no Message.Role (so Load filters them out of ByUUID). Lineage
// must walk THROUGH these gaps — the latest user turn often has an
// attachment as its direct parent, with the previous chat turn 2-3
// hops up. Bug history: real 4174-message transcript was returning
// only 4-message lineage because the chain broke at the first
// filtered ancestor.
func TestLineageWalksThroughAttachments(t *testing.T) {
	tr, err := Load("testdata/with-attachments.jsonl")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := tr.Lineage("a-2")
	want := []string{"u-1", "a-1", "u-2", "a-2"}
	if len(got) != len(want) {
		t.Fatalf("Lineage len = %d, want %d (chain broke at attachment)", len(got), len(want))
	}
	for i, m := range got {
		if m.UUID != want[i] {
			t.Errorf("Lineage[%d].UUID = %q, want %q", i, m.UUID, want[i])
		}
	}
}

func TestLoadDir(t *testing.T) {
	tr, err := LoadDir("testdata")
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	// Unique UUIDs across all fixtures (duplicates deduplicated):
	// linear: u-1, a-1, u-2 (3)
	// one-branch adds: u-2a, u-2b (2 new; u-1, a-1 are dupes)
	// multi-branch adds: u-3 (1 new; a-1 is a dupe)
	// truncated: u-1, a-1 (both dupes, 0 new)
	// Total unique = 6
	if got := len(tr.ByUUID); got < 6 {
		t.Errorf("LoadDir ByUUID len = %d, want >= 6", got)
	}
	// a-1 should have multiple children across files (u-2 from linear,
	// u-2a/u-2b from one-branch, u-3 from multi-branch).
	if got := len(tr.Children["a-1"]); got < 3 {
		t.Errorf("Children[a-1] = %v, want >= 3", tr.Children["a-1"])
	}
}
