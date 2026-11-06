package setup

import (
	"context"

	"github.com/zadykian/ws/step"
)

// The states that a report's items share.
const (
	stateMissing = "missing"
)

// fixed is items that a step has read already.
type fixed []step.Item

// Report returns the items.
func (f fixed) Report(context.Context) ([]step.Item, error) { return f, nil }

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
