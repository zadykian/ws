package userfile

import (
	"context"
	"errors"
	"fmt"
	"github.com/zadykian/ws/engine/core/resource/file"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// errNotDir is a plan's error for a path that holds no directory: a step stops there, and its
// report shows a conflict.
var errNotDir = errors.New("is not a directory")

// Dir is a directory of the user's that ws writes under, as File is a file. Its plan and
// its report read it alike, from one stat.
type Dir struct {
	M    effect.Machine
	Out  effect.Out
	Path string
	Mode fs.FileMode
	// Fix says whether a run sets Mode where the mode differs. Otherwise a run makes the directory
	// where missing and leaves the mode of one that exists.
	Fix bool
}

// read stats the directory. A missing path is not an error. found is set for a path that exists,
// and info says where a link points. It returns errNotDir for a path that is no directory.
func (d Dir) read() (found Found, info effect.Info, err error) {
	info, err = d.M.Stat(d.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Found{}, info, nil
	}
	if err != nil {
		return Found{}, info, err
	}
	found = Found{Exists: true, Mode: info.Mode}
	if !info.Mode.IsDir() {
		return found, info, fmt.Errorf("%s %w", d.Path, errNotDir)
	}
	return found, info, nil
}

// change returns the change that brings the directory, as read in found, to its mode, or none.
func (d Dir) change(found Found) []step.Change {
	switch {
	case !found.Exists:
		return []step.Change{file.UserMkdir{M: d.M, Path: d.Path, Mode: d.Mode}}
	case d.Fix && found.Mode.Perm() != d.Mode:
		return []step.Change{file.UserChmod{M: d.M, Path: d.Path, Mode: d.Mode}}
	}
	return nil
}

// Report says how the directory stands, as a plan reads it. A path that is no directory shows as
// a conflict, where a plan stops.
func (d Dir) Report(context.Context) ([]step.Item, error) {
	found, _, err := d.read()
	p := Path{Kind: step.DirItem, Path: d.Path}
	if d.Fix {
		p.Mode = d.Mode
	}
	switch {
	case errors.Is(err, errNotDir):
		item := p.itemOf(found)
		item.Found += ", no directory"
		item.Conflict = "not a directory: move it away, and the next run makes the directory"
		return []step.Item{item}, nil
	case err != nil:
		return nil, err
	}
	return []step.Item{p.itemOf(found)}, nil
}

// Dirs is directories in the order that a run makes them, parents first.
type Dirs []Dir

// DirLink is a directory that is a link: where a step writes under it, it warns once.
type DirLink struct{ Path, To string }

// Warn says that the step writes under the link, where it plans writes. goes names what lands there.
func (l DirLink) Warn(out effect.Out, writes int, goes string) {
	if l.Path != "" && writes > 0 {
		out.Warn("%s is a link to %s: %s", l.Path, l.To, goes)
	}
}

// Plan returns the changes that make the directories, and the last of them that is a link. It stops
// at a path that is no directory, and at any error other than a missing path.
func (ds Dirs) Plan() ([]step.Change, DirLink, error) {
	var cs []step.Change
	var link DirLink
	for _, d := range ds {
		found, info, err := d.read()
		if err != nil {
			return nil, DirLink{}, err
		}
		if info.Link != "" {
			link = DirLink{d.Path, info.Link}
		}
		cs = append(cs, d.change(found)...)
	}
	return cs, link, nil
}

// Report joins the directories' items.
func (ds Dirs) Report(ctx context.Context) ([]step.Item, error) {
	parts := make([]step.Reporter, len(ds))
	for i, d := range ds {
		parts[i] = d
	}
	return Gather(ctx, parts...)
}

// WsDirs is ~/.config and ws's directory in it, which git, zsh and the tunnel write under. A run
// sets the mode of ws's directory, which holds mac.conf.
func WsDirs(m effect.Machine, out effect.Out, home string) Dirs {
	return Dirs{
		{M: m, Out: out, Path: home + "/.config", Mode: 0o755},
		{M: m, Out: out, Path: home + "/.config/ws", Mode: 0o700, Fix: true},
	}
}
