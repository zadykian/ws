package host

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
)

// The states that a report's items share.
const (
	missing = "missing"
	present = "present"
)

// attrsChanged says how a path stands against ws where its content, if it has any, is right.
const attrsChanged = "its owner or mode changed"

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
