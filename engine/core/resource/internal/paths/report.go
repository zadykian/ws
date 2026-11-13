// Package paths holds what the resource packages share: the states of a report, the reading of a
// file as root and the guards of a user's home.
package paths

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
)

// The states that a report's items share.
const (
	Missing = "missing"
	Present = "present"
)

// AttrsChanged says how a path stands against ws where its content, if it has any, is right.
const AttrsChanged = "its owner or mode changed"

// AttrsText says a path's owner and mode, as a report shows them.
func AttrsText(a effect.Attrs) string {
	return fmt.Sprintf("owner %d:%d, mode %o", a.UID, a.GID, a.Mode)
}

// ReadRoot reads a file and its attributes as root. ok is false where the file is missing.
func ReadRoot(m effect.Machine, path string) (body []byte, a effect.Attrs, ok bool, err error) {
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
