// cc-rich/internal/actions/quote.go
package actions

import (
	"fmt"
	"os"
	"time"
)

// QuoteEntry is what F4 appends to the buffer.
type QuoteEntry struct {
	SessionID string
	MsgUUID   string
	Timestamp time.Time
	Text      string
}

// QuoteToBuffer appends a citation block to bufferPath. Creates the file
// if missing. Atomic: writes to a tmp neighbor + rename.
func QuoteToBuffer(bufferPath string, e QuoteEntry) error {
	existing, _ := os.ReadFile(bufferPath) // ignore not-found
	header := fmt.Sprintf("\n// quoted from session %s : msg %s @ %s\n", e.SessionID, e.MsgUUID, e.Timestamp.Format(time.RFC3339))
	body := "> " + e.Text + "\n\n---\n"
	combined := append(existing, []byte(header+body)...)

	tmp := bufferPath + ".tmp"
	if err := os.WriteFile(tmp, combined, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, bufferPath)
}
