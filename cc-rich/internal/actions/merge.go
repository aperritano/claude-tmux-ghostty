// cc-rich/internal/actions/merge.go
package actions

import (
	"fmt"
	"os"
	"strings"
)

// Citation is one message imported via the merge composer.
type Citation struct {
	SourceSID string
	MsgUUID   string
	Text      string
}

// WriteMergeBuffer atomically writes a merge artifact to targetPath.
// Format: each citation as "// imported from branch <sid>:msg<uuid>"
// followed by the text as a Markdown blockquote.
func WriteMergeBuffer(targetPath string, cites []Citation) error {
	if len(cites) == 0 {
		return fmt.Errorf("WriteMergeBuffer: no citations")
	}
	var sb strings.Builder
	for _, c := range cites {
		fmt.Fprintf(&sb, "// imported from branch %s:msg%s\n\n", c.SourceSID, c.MsgUUID)
		for _, line := range strings.Split(c.Text, "\n") {
			fmt.Fprintf(&sb, "> %s\n", line)
		}
		sb.WriteString("\n")
	}
	tmp := targetPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, targetPath)
}
