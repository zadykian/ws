package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// The states that a report's items share.
const (
	stateMissing   = "missing"
	statePresent   = "present"
	stateInstalled = "installed"
)

// path is what a step read of one path: whether it exists, its mode and its text.
type path struct {
	Exists bool
	Mode   fs.FileMode
	Text   []byte
}

// writtenBy starts the line at the top of a file that ws writes on the Mac, after its format's
// comment mark.
var writtenBy = []byte("Written by ")

// gather joins the items of the parts, and keeps those before a failure, as host.Group does.
func gather(ctx context.Context, parts ...step.Reporter) ([]step.Item, error) {
	var items []step.Item
	for _, p := range parts {
		got, err := p.Report(ctx)
		items = append(items, got...)
		if err != nil {
			return items, err
		}
	}
	return items, nil
}

// fixed is items that a step has read already.
type fixed []step.Item

// Report returns the items.
func (f fixed) Report(context.Context) ([]step.Item, error) { return f, nil }

func modeText(mode fs.FileMode) string { return fmt.Sprintf("mode %o", mode.Perm()) }

// userPath is a path of the user's that a report never reads inside. A Mode of 0 leaves the mode
// out, as a run leaves it to the user there.
type userPath struct {
	M    effect.Machine
	Kind step.ItemKind
	Path string
	Mode fs.FileMode
}

// itemOf says how the path stands, as read in found.
func (p userPath) itemOf(found path) step.Item {
	item := step.Item{Kind: p.Kind, Name: p.Path, Want: statePresent, Found: stateMissing}
	if p.Mode != 0 {
		item.Want = modeText(p.Mode)
	}
	if !found.Exists {
		return item
	}
	item.Found = statePresent
	if p.Mode != 0 {
		item.Found = modeText(found.Mode)
	}
	if item.Found != item.Want {
		item.Conflict = "its mode changed"
	}
	return item
}

// Report reads the path.
func (p userPath) Report(context.Context) ([]step.Item, error) {
	info, err := p.M.Stat(p.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return []step.Item{p.itemOf(path{})}, nil
	}
	if err != nil {
		return nil, err
	}
	return []step.Item{p.itemOf(path{Exists: true, Mode: info.Mode})}, nil
}

// userFile is a file of the user's that ws writes, as host.File is one of root's. Its plan and its
// report read it alike, so that a report stands off where a plan changes the file or leaves it.
type userFile struct {
	M    effect.Machine
	Path string
	Text []byte
	// Mode is the file's mode. 0 leaves it to the user until a write, which sets 644.
	Mode fs.FileMode
	// Own says whether a run replaces a file that holds these bytes. Nil means that it always does.
	Own func(have []byte) bool
	// Binary keeps a program out of the diff.
	Binary bool
}

// errNotWs is a plan's error for a file that is not ws's: its step warns of it, or stops.
var errNotWs = errors.New("not ws's")

// read reads the file. ours is false where Own refuses the text, and where Own is set and the
// path is no file, such as a link to nowhere: a run replaces neither.
func (f userFile) read() (found path, ours bool, err error) {
	info, err := f.M.Stat(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return path{}, true, nil
	}
	if err != nil {
		return path{}, false, err
	}
	found = path{Exists: true, Mode: info.Mode}
	nowhere := info.Link != "" && info.Mode == 0
	if f.Own != nil && (nowhere || !info.Mode.IsRegular()) {
		return found, false, nil
	}
	if found.Text, err = f.M.ReadFile(f.Path); err != nil {
		return found, false, fmt.Errorf("cannot read %s: %w", f.Path, err)
	}
	return found, f.Own == nil || f.Own(found.Text), nil
}

// change returns the change that brings the file, as read in found, to Text and Mode, or none.
func (f userFile) change(found path) []step.Change {
	mode := f.Mode
	if mode == 0 {
		mode = 0o644
	}
	switch {
	case !found.Exists || !bytes.Equal(found.Text, f.Text):
		return []step.Change{Write{f.Path, found.Text, f.Text, found.Exists, mode}}
	case f.Mode != 0 && found.Mode.Perm() != f.Mode:
		return []step.Change{SetMode{f.Path, f.Mode}}
	}
	return nil
}

// plan returns a write, a fix of the mode, or nothing. It returns errNotWs, and no change, where
// the file is not ws's.
func (f userFile) plan() ([]step.Change, error) {
	found, ours, err := f.read()
	if err != nil {
		return nil, err
	}
	if !ours {
		return nil, errNotWs
	}
	return f.change(found), nil
}

// Report compares the file as a plan does. A file that isn't ws's shows no diff: a run leaves it,
// --check shows none, and it may hold anything.
func (f userFile) Report(context.Context) ([]step.Item, error) {
	found, ours, err := f.read()
	if err != nil {
		return nil, err
	}
	item := userPath{Kind: step.FileItem, Path: f.Path, Mode: f.Mode}.itemOf(found)
	if !found.Exists {
		return []step.Item{item}, nil
	}
	switch {
	case !ours:
		item.Found += ", not ws's"
		item.Conflict = "not ws's: move it away, and the next run writes ws's"
	case !bytes.Equal(found.Text, f.Text):
		item.Found += ", other content"
		item.Conflict = "differs from what ws writes: a change by hand, or another version of ws"
		if bytes.Contains(f.Text, writtenBy) && !bytes.Contains(found.Text, writtenBy) {
			item.Conflict = "holds no Written by line: ws did not write it"
		}
		if !f.Binary {
			item.Diff = effect.Diff(f.Path, found.Text, f.Text)
		}
	}
	return []step.Item{item}, nil
}

// owned plans f, and leaves a file that is not ws's, with a warning.
func owned(out effect.Out, f userFile) ([]step.Change, error) {
	cs, err := f.plan()
	if errors.Is(err, errNotWs) {
		out.Warn("%s is not ws's: move it away, and the next run writes ws's", f.Path)
		return nil, nil
	}
	return cs, err
}

// lineItem is a line that ws keeps in a file of the user's. The rest of the file stays out of the
// report, as it may hold anything.
func lineItem(path, line string, exists, holds bool) step.Item {
	item := step.Item{Kind: step.FileItem, Name: path, Want: "holds " + line, Found: stateMissing}
	switch {
	case holds:
		item.Found = item.Want
	case exists:
		item.Found = "lacks it"
	}
	return item
}
