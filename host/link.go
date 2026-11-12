package host

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Link is a symbolic link to a target. It has no owner or mode: a link's own are never used.
type Link struct {
	M      effect.Machine
	Path   string
	Target string
}

type setLink struct{ l Link }

func (c setLink) String() string { return fmt.Sprintf("link %s to %s", c.l.Path, c.l.Target) }

func (setLink) Kind() step.ChangeKind { return step.Write }

func (c setLink) Apply(context.Context) error { return c.l.M.Symlink(c.l.Target, c.l.Path) }

// Plan finds where the path points, if the path is a link.
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

// Report reads where the path points. Any other target, or no link, stands against ws.
func (l Link) Report(context.Context) ([]step.Item, error) {
	info, err := l.M.Stat(l.Path)
	conflict := ""
	if info.Link != l.Target {
		conflict = "not the link ws makes"
	}
	item := step.Item{Kind: step.LinkItem, Name: l.Path, Want: l.Target, Found: missing}
	return reportPath(item, cmp.Or(info.Link, "not a link"), conflict, err)
}
