package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
)

// Link is a symbolic link to a target. It has no owner or mode: a link's own are never used.
type Link struct {
	M      effect.Machine
	Path   string
	Target string
}

type setLink struct{ l Link }

func (c setLink) String() string { return fmt.Sprintf("link %s to %s", c.l.Path, c.l.Target) }

func (c setLink) Apply(context.Context) error { return c.l.M.Symlink(c.l.Target, c.l.Path) }

// Plan finds where the path points, if it is a link.
func (l Link) Plan(context.Context) ([]Change, error) {
	info, err := l.M.Stat(l.Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err != nil || info.Link != l.Target {
		return []Change{setLink{l}}, nil
	}
	return nil, nil
}
