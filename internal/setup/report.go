package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// The states that a report's items share.
const (
	stateMissing   = "missing"
	statePresent   = "present"
	stateInstalled = "installed"
)

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

// item says how the path stands, and whether it is there.
func (p userPath) item() (step.Item, bool, error) {
	item := step.Item{Kind: p.Kind, Name: p.Path, Want: statePresent, Found: stateMissing}
	if p.Mode != 0 {
		item.Want = modeText(p.Mode)
	}
	info, err := p.M.Stat(p.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return item, false, nil
	}
	if err != nil {
		return item, false, err
	}
	item.Found = statePresent
	if p.Mode != 0 {
		item.Found = modeText(info.Mode)
	}
	if item.Found != item.Want {
		item.Conflict = "its mode changed"
	}
	return item, true, nil
}

// Report reads the path.
func (p userPath) Report(context.Context) ([]step.Item, error) {
	item, _, err := p.item()
	if err != nil {
		return nil, err
	}
	return []step.Item{item}, nil
}

// userFile is a file of the user's that ws writes, as host.File is one of root's.
type userFile struct {
	M    effect.Machine
	Path string
	Text []byte
	Mode fs.FileMode
	// Own says whether a run replaces a file that holds these bytes. Nil means that it always does.
	Own func(have []byte) bool
	// Binary keeps a program out of the diff.
	Binary bool
}

// Report compares the file as a plan does. A file that isn't ws's shows no diff: a run leaves it,
// --check shows none, and it may hold anything.
func (f userFile) Report(context.Context) ([]step.Item, error) {
	item, there, err := userPath{f.M, step.FileItem, f.Path, f.Mode}.item()
	if err != nil {
		return nil, err
	}
	if !there {
		return []step.Item{item}, nil
	}
	have, err := f.M.ReadFile(f.Path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", f.Path, err)
	}
	switch {
	case f.Own != nil && !f.Own(have):
		item.Found += ", not ws's"
		item.Conflict = "not ws's: move it away, and the next run writes ws's"
	case !bytes.Equal(have, f.Text):
		item.Found += ", other content"
		item.Conflict = "differs from what ws writes: a change by hand, or another version of ws"
		if bytes.Contains(f.Text, writtenBy) && !bytes.Contains(have, writtenBy) {
			item.Conflict = "holds no Written by line: ws did not write it"
		}
		if !f.Binary {
			item.Diff = effect.Diff(f.Path, have, f.Text)
		}
	}
	return []step.Item{item}, nil
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
