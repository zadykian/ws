package file

import (
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/step"
)

// writtenBy is the line at the top of a file ws writes, after its format's comment mark.
var writtenBy = []byte("Written by github.com/zadykian/ws")

// reportPath returns item, with found and conflict unless err says that the path is missing.
func reportPath(item step.Item, found, conflict string, err error) ([]step.Item, error) {
	if errors.Is(err, fs.ErrNotExist) {
		return []step.Item{item}, nil
	}
	if err != nil {
		return nil, err
	}
	item.Found, item.Conflict = found, conflict
	return []step.Item{item}, nil
}
