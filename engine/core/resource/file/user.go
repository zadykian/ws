package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// The states that a report's items share.
const (
	StateMissing   = "missing"
	StatePresent   = "present"
	StateInstalled = "installed"
)

// Found is what a step read of one path: whether it exists, its mode and its text.
type Found struct {
	Exists bool
	Mode   fs.FileMode
	Text   []byte
}

// userWrittenBy starts the line at the top of a file that ws writes on the Mac, after its format's
// comment mark.
var userWrittenBy = []byte("Written by ")

// Gather joins the items of the parts, and keeps those before a failure, as host.Group does.
func Gather(ctx context.Context, parts ...step.Reporter) ([]step.Item, error) {
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

// Fixed is items that a step has read already.
type Fixed []step.Item

// Report returns the items.
func (f Fixed) Report(context.Context) ([]step.Item, error) { return f, nil }

func modeText(mode fs.FileMode) string { return fmt.Sprintf("mode %o", mode.Perm()) }

// UserPath is a path of the user's that a report never reads inside. A Mode of 0 leaves the mode
// out, as a run leaves it to the user there.
type UserPath struct {
	M    effect.Machine
	Kind step.ItemKind
	Path string
	Mode fs.FileMode
}

// itemOf says how the path stands, as read in found.
func (p UserPath) itemOf(found Found) step.Item {
	item := step.Item{Kind: p.Kind, Name: p.Path, Want: StatePresent, Found: StateMissing}
	if p.Mode != 0 {
		item.Want = modeText(p.Mode)
	}
	if !found.Exists {
		return item
	}
	item.Found = StatePresent
	if p.Mode != 0 {
		item.Found = modeText(found.Mode)
	}
	if item.Found != item.Want {
		item.Conflict = "its mode changed"
	}
	return item
}

// Report reads the path.
func (p UserPath) Report(context.Context) ([]step.Item, error) {
	info, err := p.M.Stat(p.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return []step.Item{p.itemOf(Found{})}, nil
	}
	if err != nil {
		return nil, err
	}
	return []step.Item{p.itemOf(Found{Exists: true, Mode: info.Mode})}, nil
}

// UserFile is a file of the user's that ws writes, as File is one of root's. Its plan and its
// report read it alike, so that a report stands off where a plan changes the file or leaves it.
type UserFile struct {
	M    effect.Machine
	Out  effect.Out
	Path string
	Text []byte
	// Mode is the file's mode. 0 leaves it to the user until a write, which sets 644.
	Mode fs.FileMode
	// Own says whether a run replaces a file that holds these bytes. Nil means that it always does.
	Own func(have []byte) bool
	// Binary keeps a program out of the diff.
	Binary bool
	// Warn has a plan leave a file that is not ws's with a warning, where it stops otherwise.
	Warn bool
}

// ErrNotWs is a plan's error for a file that is not ws's: its step warns of it, or stops.
var ErrNotWs = errors.New("not ws's")

// Read reads the file. ours is false where Own refuses the text, and where Own is set and the
// path is no file, such as a link to nowhere: a run replaces neither.
func (f UserFile) Read() (found Found, ours bool, err error) {
	info, err := f.M.Stat(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Found{}, true, nil
	}
	if err != nil {
		return Found{}, false, err
	}
	found = Found{Exists: true, Mode: info.Mode}
	nowhere := info.Link != "" && info.Mode == 0
	if f.Own != nil && (nowhere || !info.Mode.IsRegular()) {
		return found, false, nil
	}
	if found.Text, err = f.M.ReadFile(f.Path); err != nil {
		return found, false, fmt.Errorf("cannot read %s: %w", f.Path, err)
	}
	return found, f.Own == nil || f.Own(found.Text), nil
}

// Change returns the change that brings the file, as read in found, to Text and Mode, or none.
func (f UserFile) Change(found Found) []step.Change {
	mode := f.Mode
	if mode == 0 {
		mode = 0o644
	}
	switch {
	case !found.Exists || !bytes.Equal(found.Text, f.Text):
		return []step.Change{UserWrite{M: f.M, Out: f.Out, Path: f.Path, Before: found.Text, After: f.Text, Exists: found.Exists, Mode: mode}}
	case f.Mode != 0 && found.Mode.Perm() != f.Mode:
		return []step.Change{UserChmod{M: f.M, Path: f.Path, Mode: f.Mode}}
	}
	return nil
}

// Plan returns a write, a fix of the mode, or nothing. Where the file is not ws's, it plans nothing:
// it warns where Warn is set, and returns ErrNotWs otherwise.
func (f UserFile) Plan(context.Context) ([]resource.Change, error) {
	found, ours, err := f.Read()
	switch {
	case err != nil:
		return nil, err
	case ours:
		return applying(f.Change(found)), nil
	case f.Warn:
		f.Out.Warn("%s is not ws's: move it away, and the next run writes ws's", f.Path)
		return nil, nil
	}
	return nil, ErrNotWs
}

// Report compares the file as a plan does. A file that isn't ws's shows no diff: a run leaves it,
// --check shows none, and it may hold anything.
func (f UserFile) Report(context.Context) ([]step.Item, error) {
	found, ours, err := f.Read()
	if err != nil {
		return nil, err
	}
	item := UserPath{Kind: step.FileItem, Path: f.Path, Mode: f.Mode}.itemOf(found)
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
		if bytes.Contains(f.Text, userWrittenBy) && !bytes.Contains(found.Text, userWrittenBy) {
			item.Conflict = "holds no Written by line: ws did not write it"
		}
		if !f.Binary {
			item.Diff = effect.Diff(f.Path, found.Text, f.Text)
		}
	}
	return []step.Item{item}, nil
}

// lineItem is a line that ws keeps in a file of the user's. The rest of the file stays out of the
// report, as it may hold anything.
func lineItem(path, line string, exists, holds bool) step.Item {
	item := step.Item{Kind: step.FileItem, Name: path, Want: "holds " + line, Found: StateMissing}
	switch {
	case holds:
		item.Found = item.Want
	case exists:
		item.Found = "lacks it"
	}
	return item
}
