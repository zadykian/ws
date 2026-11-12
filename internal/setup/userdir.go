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
