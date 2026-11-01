package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
)

// Dir is a directory with its mode and owner. Its parent must exist: list the parents first.
type Dir struct {
	M    effect.Machine
	Path string
	Mode fs.FileMode
	// UID and GID are the owner, root's by default.
	UID, GID int
}

type setDir struct {
	d      Dir
	exists bool
}

func (c setDir) String() string {
	if !c.exists {
		return fmt.Sprintf("create directory %s, owner %d:%d, mode %o", c.d.Path, c.d.UID, c.d.GID, c.d.Mode.Perm())
	}
	return fmt.Sprintf("set owner %d:%d and mode %o of %s", c.d.UID, c.d.GID, c.d.Mode.Perm(), c.d.Path)
}

func (c setDir) Apply(context.Context) error {
	if !c.exists {
		if err := c.d.M.Mkdir(c.d.Path, c.d.Mode); err != nil {
			return err
		}
	}
	if err := c.d.M.Chown(c.d.Path, c.d.UID, c.d.GID); err != nil {
		return err
	}
	return c.d.M.Chmod(c.d.Path, c.d.Mode)
}

// Plan reads the directory's owner and mode.
func (d Dir) Plan(context.Context) ([]Change, error) {
	got, err := d.M.RootAttrs(d.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return []Change{setDir{d: d}}, nil
	}
	if err != nil {
		return nil, err
	}
	if got.UID != d.UID || got.GID != d.GID || got.Mode != d.Mode.Perm() {
		return []Change{setDir{d, true}}, nil
	}
	return nil, nil
}
