// cc-rich/internal/sessiontree/tree.go
// Types, constructor, and loading for the parent/child message tree.
package sessiontree

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
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
}

// Tree is the loaded shape of one or more transcripts: a flat index plus
// child fan-out by parent UUID.
type Tree struct {
	ByUUID   map[string]*Message
	Children map[string][]string // parent UUID -> list of child UUIDs (insertion order)
	Roots    []string            // UUIDs with no in-tree parent
}

// New returns an empty Tree ready for population by Load (Task 2.2).
func New() *Tree {
	return &Tree{
		ByUUID:   make(map[string]*Message),
		Children: make(map[string][]string),
	}
}

type rawTurn struct {
	UUID       string `json:"uuid"`
	ParentUUID string `json:"parentUuid"`
	Timestamp  string `json:"timestamp"`
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
// Unparseable lines are skipped silently.
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
		}
		tr.ByUUID[m.UUID] = m
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	// Build Children + Roots once all messages are loaded.
	for uuid, m := range tr.ByUUID {
		if m.ParentUUID == "" || tr.ByUUID[m.ParentUUID] == nil {
			tr.Roots = append(tr.Roots, uuid)
			continue
		}
		tr.Children[m.ParentUUID] = append(tr.Children[m.ParentUUID], uuid)
	}
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
	sortStrings(out)
	return out
}

// sortStrings is a local copy to avoid pulling in "sort" everywhere.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// Lineage returns the chain of messages from the earliest ancestor down to
// targetUUID, inclusive. Empty slice if targetUUID is not in the tree.
func (t *Tree) Lineage(targetUUID string) []*Message {
	var chain []*Message
	cur := t.ByUUID[targetUUID]
	for cur != nil {
		chain = append([]*Message{cur}, chain...)
		cur = t.ByUUID[cur.ParentUUID]
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
	// Rebuild Children + Roots from the merged ByUUID.
	for uuid, m := range tr.ByUUID {
		if m.ParentUUID == "" || tr.ByUUID[m.ParentUUID] == nil {
			tr.Roots = append(tr.Roots, uuid)
			continue
		}
		tr.Children[m.ParentUUID] = append(tr.Children[m.ParentUUID], uuid)
	}
	return tr, nil
}
