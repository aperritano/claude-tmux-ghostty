// cc-rich/internal/sessiontree/tree.go
// Types and constructor for the parent/child message tree.
package sessiontree

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Block is one piece of a message's content (text, tool_use, tool_result, thinking).
type Block struct {
	Type string // "text" | "tool_use" | "tool_result" | "thinking"
	Text string // raw text for "text" or "thinking"; JSON dump for tool_use input
}

// Usage records token-accounting fields from a transcript turn.
type Usage struct {
	InputTokens              int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
	OutputTokens             int
}

// Message is one turn in a Claude Code session.
type Message struct {
	UUID       string
	ParentUUID string
	Role       string // "user" | "assistant"
	Content    []Block
	Timestamp  time.Time
	Model      string
	Usage      Usage
	SourceFile string // absolute path of the JSONL this came from
	Cwd        string // working directory when the turn was issued
}

// Tree is the loaded shape of one or more transcripts: a flat index plus
// child fan-out by parent UUID.
type Tree struct {
	ByUUID   map[string]*Message
	Children map[string][]string // parent UUID -> list of child UUIDs (insertion order)
	Roots    []string            // UUIDs with no in-tree parent
	// ParentLinks records uuid -> parentUuid for EVERY parsed line in
	// the JSONL, including attachments and metadata that were filtered
	// out of ByUUID (no Message.Role). Lineage uses this to walk past
	// non-chat ancestors so the chain doesn't break at filtered nodes.
	ParentLinks map[string]string
}

// New returns an empty Tree.
func New() *Tree {
	return &Tree{
		ByUUID:      make(map[string]*Message),
		Children:    make(map[string][]string),
		ParentLinks: make(map[string]string),
	}
}

// buildIndices populates tr.Children and tr.Roots from tr.ByUUID. Idempotent
// — callers may invoke after merging multiple loads. Resets Children and
// Roots first so consecutive calls don't accumulate duplicates.
func buildIndices(t *Tree) {
	t.Children = make(map[string][]string)
	t.Roots = t.Roots[:0]
	for uuid, m := range t.ByUUID {
		if m.ParentUUID == "" || t.ByUUID[m.ParentUUID] == nil {
			t.Roots = append(t.Roots, uuid)
			continue
		}
		t.Children[m.ParentUUID] = append(t.Children[m.ParentUUID], uuid)
	}
}

type rawTurn struct {
	UUID       string `json:"uuid"`
	ParentUUID string `json:"parentUuid"`
	Timestamp  string `json:"timestamp"`
	Cwd        string `json:"cwd,omitempty"`
	Message    struct {
		Role    string     `json:"role"`
		Model   string     `json:"model"`
		Content []rawBlock `json:"content"`
		Usage   *rawUsage  `json:"usage,omitempty"`
	} `json:"message"`
}

type rawBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Input json.RawMessage `json:"input,omitempty"` // tool_use payload
}

type rawUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// Load reads a JSONL transcript and returns a single-file Tree.
// Unparseable lines are skipped silently. TODO: future work — return a
// counter or accept an io.Writer for observability of corrupt records.
func Load(path string) (*Tree, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tr := New()
	sc := bufio.NewScanner(f)
	// Transcripts can have very long lines (large tool outputs).
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rt rawTurn
		if err := json.Unmarshal(line, &rt); err != nil {
			continue // skip unparseable line
		}
		// Track parent links for ALL UUIDs (even filtered metadata) so
		// Lineage can walk past attachment/metadata ancestors. Real
		// transcripts intersperse user/assistant turns with attachment
		// records — without this, the chain breaks at the first
		// filtered ancestor and Lineage returns a 2-4 message slice of
		// a 4000-message conversation.
		if rt.UUID != "" {
			tr.ParentLinks[rt.UUID] = rt.ParentUUID
		}
		if rt.UUID == "" || rt.Message.Role == "" {
			continue // skip metadata lines (last-prompt, permission-mode, etc.)
		}
		ts, _ := time.Parse(time.RFC3339Nano, rt.Timestamp)
		blocks := make([]Block, 0, len(rt.Message.Content))
		for _, b := range rt.Message.Content {
			block := Block{Type: b.Type, Text: b.Text}
			if b.Type == "tool_use" && len(b.Input) > 0 {
				block.Text = string(b.Input)
			}
			blocks = append(blocks, block)
		}
		var usage Usage
		if rt.Message.Usage != nil {
			usage = Usage{
				InputTokens:              rt.Message.Usage.InputTokens,
				CacheCreationInputTokens: rt.Message.Usage.CacheCreationInputTokens,
				CacheReadInputTokens:     rt.Message.Usage.CacheReadInputTokens,
				OutputTokens:             rt.Message.Usage.OutputTokens,
			}
		}
		m := &Message{
			UUID:       rt.UUID,
			ParentUUID: rt.ParentUUID,
			Role:       rt.Message.Role,
			Content:    blocks,
			Timestamp:  ts,
			Model:      rt.Message.Model,
			Usage:      usage,
			SourceFile: path,
			Cwd:        rt.Cwd,
		}
		tr.ByUUID[m.UUID] = m
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	buildIndices(tr)
	return tr, nil
}

// BranchPoints returns UUIDs of messages that have more than one direct child.
// Order is stable (sorted ascending) for testability.
func (t *Tree) BranchPoints() []string {
	var out []string
	for parent, kids := range t.Children {
		if len(kids) > 1 {
			out = append(out, parent)
		}
	}
	sort.Strings(out)
	return out
}

// AllByTime returns every message in the tree sorted by ascending
// timestamp. Used by the conversation view to show the full
// transcript including sibling branches (sub-agent dispatches,
// parallel tool turns) that aren't on any single Lineage chain.
//
// Compare to Lineage(uuid) which walks one branch back from a target —
// useful for "show what led to THIS turn" but misses concurrent
// branches in multi-agent sessions.
func (t *Tree) AllByTime() []*Message {
	out := make([]*Message, 0, len(t.ByUUID))
	for _, m := range t.ByUUID {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.Before(out[j].Timestamp)
	})
	return out
}

// Lineage returns the chain of messages from the earliest ancestor down to
// targetUUID, inclusive. Empty slice if targetUUID is not in the tree.
//
// Walks t.ParentLinks (which records parents for ALL parsed UUIDs, not
// just chat messages) so the chain can step past attachment / metadata
// ancestors that Load filtered out of ByUUID. Only nodes present in
// ByUUID are emitted into the returned chain.
func (t *Tree) Lineage(targetUUID string) []*Message {
	var chain []*Message
	uuid := targetUUID
	for uuid != "" {
		if m := t.ByUUID[uuid]; m != nil {
			chain = append([]*Message{m}, chain...)
		}
		next, ok := t.ParentLinks[uuid]
		if !ok {
			break // reached a UUID we don't have any record of
		}
		uuid = next
	}
	return chain
}

// LoadDir loads every *.jsonl in dir into a single shared Tree. Useful for
// finding sibling sessions that descend from the same parent UUID.
func LoadDir(dir string) (*Tree, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	tr := New()
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		sub, err := Load(path)
		if err != nil {
			continue
		}
		for uuid, m := range sub.ByUUID {
			if _, exists := tr.ByUUID[uuid]; !exists {
				tr.ByUUID[uuid] = m
			}
		}
	}
	buildIndices(tr)
	return tr, nil
}
