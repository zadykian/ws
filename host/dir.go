package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Dir is a directory with its mode and owner. Its parent must exist: list the parents first.
type Dir struct {
	M    effect.Machine
	Path string
	Mode fs.FileMode
	// UID and GID are the owner, root's by default.
	UID, GID int
	// CreateOnly leaves an existing directory alone, whatever its mode and owner.
	CreateOnly bool
	// home, set by Account.Dir alone, is the home that the path lies below.
	home string
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

func (setDir) Kind() step.ChangeKind { return step.Write }

// Apply looks for a link on the way again, as writeFile's does.
func (c setDir) Apply(context.Context) error {
	if err := c.d.guard(); err != nil {
		return err
	}
	if err := makeParents(c.d.M, c.d.home, c.d.Path, c.d.UID, c.d.GID); err != nil {
		return err
	}
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

// guard refuses a link on the way to a home directory. Another Dir is not checked.
func (d Dir) guard() error {
	if d.home == "" {
		return nil
	}
	return guardHome(d.M, d.home, d.Path)
}

// Plan reads the directory's owner and mode.
func (d Dir) Plan(context.Context) ([]Change, error) {
	if err := d.guard(); err != nil {
		return nil, err
	}
	got, err := d.M.RootAttrs(d.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return []Change{setDir{d: d}}, nil
	}
	if err != nil {
		return nil, err
	}
	if d.CreateOnly {
		return nil, nil
	}
	if got.UID != d.UID || got.GID != d.GID || got.Mode != d.Mode.Perm() {
		return []Change{setDir{d, true}}, nil
	}
	return nil, nil
}

// Report reads the directory's owner and mode, which stand against ws where Plan would fix them.
func (d Dir) Report(context.Context) ([]step.Item, error) {
	if err := d.guard(); err != nil {
		return nil, err
	}
	attrs := effect.Attrs{UID: d.UID, GID: d.GID, Mode: d.Mode.Perm()}
	want, conflict := attrsText(attrs), ""
	got, err := d.M.RootAttrs(d.Path)
	switch {
	case d.CreateOnly:
		want = present
	case got != attrs:
		conflict = attrsChanged
	}
	item := step.Item{Kind: step.DirItem, Name: d.Path, Want: want, Found: missing}
	return reportPath(item, attrsText(got), conflict, err)
}
