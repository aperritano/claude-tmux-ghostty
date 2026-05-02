// cc-rich/internal/sessiontree/tree.go
// Types and constructor for the parent/child message tree. Loading is
// implemented in load.go (added by Task 2.2).
package sessiontree

import "time"

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
