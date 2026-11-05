package host

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// The states that a report's items share.
const (
	missing = "missing"
	present = "present"
)

// attrsChanged says how a path stands against ws where its content, if it has any, is right.
const attrsChanged = "its owner or mode changed"

// writtenBy is the line at the top of a file ws writes, after its format's comment mark.
var writtenBy = []byte("Written by github.com/zadykian/ws")

func attrsText(a effect.Attrs) string {
	return fmt.Sprintf("owner %d:%d, mode %o", a.UID, a.GID, a.Mode)
}

// readRoot reads a file and its attributes as root. ok is false where the file is missing.
func readRoot(m effect.Machine, path string) (body []byte, a effect.Attrs, ok bool, err error) {
	body, err = m.ReadRootFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, a, false, nil
	}
	if err != nil {
		return nil, a, false, err
	}
	a, err = m.RootAttrs(path)
	return body, a, err == nil, err
}

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
