package file

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// errNotDir is a plan's error for a path that holds no directory: a step stops there, and its
// report shows a conflict.
var errNotDir = errors.New("is not a directory")

// UserDir is a directory of the user's that ws writes under, as UserFile is a file. Its plan and
// its report read it alike, from one stat.
type UserDir struct {
	M    effect.Machine
	Out  effect.Out
	Path string
	Mode fs.FileMode
}

// read stats the directory. A missing path is not an error. found is set for a path that exists,
// and info says where a link points. It returns errNotDir for a path that is no directory.
func (d UserDir) read() (found Found, info effect.Info, err error) {
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

// change returns the change that makes the directory where missing, or none. A run leaves the mode
// of a directory that exists.
func (d UserDir) change(found Found) []step.Change {
	if !found.Exists {
		return []step.Change{UserMkdir{M: d.M, Path: d.Path, Mode: d.Mode}}
	}
	return nil
}

// Plan makes the directory where missing. It stops at a path that is no directory.
func (d UserDir) Plan(context.Context) ([]resource.Change, error) {
	found, _, err := d.read()
	if err != nil {
		return nil, err
	}
	return applying(d.change(found)), nil
}

// Report says how the directory stands, as a plan reads it. A path that is no directory shows as
// a conflict, where a plan stops.
func (d UserDir) Report(context.Context) ([]step.Item, error) {
	found, _, err := d.read()
	p := UserPath{Kind: step.DirItem, Path: d.Path}
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

// UserDirs is directories in the order that a run makes them, parents first.
type UserDirs []UserDir

// UserDirLink is a directory that is a link: where a step writes under it, it warns once.
type UserDirLink struct{ Path, To string }

// Warn says that the step writes under the link, where it plans writes. goes names what lands there.
func (l UserDirLink) Warn(out effect.Out, writes int, goes string) {
	if l.Path != "" && writes > 0 {
		out.Warn("%s is a link to %s: %s", l.Path, l.To, goes)
	}
}

// Plan returns the changes that make the directories, and the last of them that is a link. It stops
// at a path that is no directory, and at any error other than a missing path.
func (ds UserDirs) Plan() ([]step.Change, UserDirLink, error) {
	var cs []step.Change
	var link UserDirLink
	for _, d := range ds {
		found, info, err := d.read()
		if err != nil {
			return nil, UserDirLink{}, err
		}
		if info.Link != "" {
			link = UserDirLink{d.Path, info.Link}
		}
		cs = append(cs, d.change(found)...)
	}
	return cs, link, nil
}

// Report joins the directories' items.
func (ds UserDirs) Report(ctx context.Context) ([]step.Item, error) {
	parts := make([]step.Reporter, len(ds))
	for i, d := range ds {
		parts[i] = d
	}
	return Gather(ctx, parts...)
}
