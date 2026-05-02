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
	tr, _ := Load("testdata/one-branch.jsonl")
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
