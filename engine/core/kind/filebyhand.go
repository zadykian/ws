package kind

import (
	"errors"
	"fmt"
)

// fileByHand is a file's fields for a file that its owner edits by hand too, as a user's own
// config is.
type fileByHand struct {
	// LineAt puts Line at the file's "top", the default, or at its "end", where a format reads its
	// last setting last.
	LineAt string `toml:"line_at"`
}

// checkByHand refuses the fields of fileByHand that no plan could take.
func (f filePart) checkByHand() error {
	switch {
	case f.LineAt != "" && f.Line == "":
		return errors.New("line_at goes with line")
	case f.LineAt != "" && f.LineAt != "top" && f.LineAt != "end":
		return fmt.Errorf("line_at %q is top or end", f.LineAt)
	}
	return nil
}
