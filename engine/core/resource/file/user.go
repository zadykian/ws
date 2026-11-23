package file

import (
	"io/fs"

	"github.com/zadykian/ws/engine/core/step"
)

// The states that a report's items share.
const (
	StateMissing = "missing"
	StatePresent = "present"
)

// Found is what a step read of one path: whether it exists, its mode and its text.
type Found struct {
	Exists bool
	Mode   fs.FileMode
	Text   []byte
}

// lineItem is a line that ws keeps in a file of the user's. The rest of the file stays out of the
// report, as it may hold anything.
func lineItem(path, line string, exists, holds bool) step.Item {
	item := step.Item{Kind: step.FileItem, Name: path, Want: "holds " + line, Found: StateMissing}
	switch {
	case holds:
		item.Found = item.Want
	case exists:
		item.Found = "lacks it"
	}
	return item
}
