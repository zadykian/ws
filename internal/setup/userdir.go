package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// errNotDir is a plan's error for a path that holds no directory: a step stops there, and its
// report shows a conflict.
var errNotDir = errors.New("is not a directory")

// userDir is a directory of the user's that ws writes under, as userFile is a file. Its plan and
// its report read it alike, from one stat.
type userDir struct {
	M    effect.Machine
	Path string
	Mode fs.FileMode
	// Fix says whether a run sets Mode where the mode differs. Otherwise a run makes the directory
	// where missing and leaves the mode of one that exists.
	Fix bool
}

// read stats the directory. A missing path is not an error. found is set for a path that exists,
// and info says where a link points. It returns errNotDir for a path that is no directory.
func (d userDir) read() (found path, info effect.Info, err error) {
	info, err = d.M.Stat(d.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return path{}, info, nil
	}
	if err != nil {
		return path{}, info, err
	}
	found = path{Exists: true, Mode: info.Mode}
	if !info.Mode.IsDir() {
		return found, info, fmt.Errorf("%s %w", d.Path, errNotDir)
	}
	return found, info, nil
}

// change returns the change that brings the directory, as read in found, to its mode, or none.
func (d userDir) change(found path) []step.Change {
	switch {
	case !found.Exists:
		return []step.Change{MakeDir{d.Path, d.Mode}}
	case d.Fix && found.Mode.Perm() != d.Mode:
		return []step.Change{SetMode{d.Path, d.Mode}}
	}
	return nil
}

// Report says how the directory stands, as a plan reads it. A path that is no directory shows as
// a conflict, where a plan stops.
func (d userDir) Report(context.Context) ([]step.Item, error) {
	found, _, err := d.read()
	p := userPath{Kind: step.DirItem, Path: d.Path}
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

// userDirs is directories in the order that a run makes them, parents first.
type userDirs []userDir

// dirLink is a directory that is a link: where a step writes under it, it warns once.
type dirLink struct{ Path, To string }

// warn says that the step writes under the link, where it plans writes.
func (l dirLink) warn(out effect.Out, writes int) {
	if l.Path != "" && writes > 0 {
		out.Warn("%s is a link to %s: ws's files go there", l.Path, l.To)
	}
}

// plan returns the changes that make the directories, and the last of them that is a link. It stops
// at a path that is no directory, and at any error other than a missing path.
func (ds userDirs) plan() ([]step.Change, dirLink, error) {
	var cs []step.Change
	var link dirLink
	for _, d := range ds {
		found, info, err := d.read()
		if err != nil {
			return nil, dirLink{}, err
		}
		if info.Link != "" {
			link = dirLink{d.Path, info.Link}
		}
		cs = append(cs, d.change(found)...)
	}
	return cs, link, nil
}

// Report joins the directories' items.
func (ds userDirs) Report(ctx context.Context) ([]step.Item, error) {
	parts := make([]step.Reporter, len(ds))
	for i, d := range ds {
		parts[i] = d
	}
	return gather(ctx, parts...)
}

// wsDirs is ~/.config and ws's directory in it, which git, zsh and the tunnel write under. A run
// sets the mode of ws's directory, which holds mac.conf.
func wsDirs(m effect.Machine, home string) userDirs {
	return userDirs{
		{M: m, Path: home + "/.config", Mode: 0o755},
		{M: m, Path: home + "/.config/ws", Mode: 0o700, Fix: true},
	}
}
