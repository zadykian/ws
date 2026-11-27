package file

import (
	"io/fs"
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
