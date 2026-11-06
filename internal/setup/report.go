package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// The states that a report's items share.
const (
	stateMissing = "missing"
	statePresent = "present"
)

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
