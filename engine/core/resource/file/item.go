package file

import (
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/step"
)

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
