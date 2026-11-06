package tui

import (
	"slices"

	"github.com/zadykian/ws/step"
)

// learned returns the list with a row for e's step, where a stage runs it and no scan named it. A
// new Mac has no mac.conf to name a stage's steps by, and no role of ws has steps before ws is there.
func (l pickList) learned(e step.Event) pickList {
	if e.Parent == "" || l.place(e) >= 0 {
		return l
	}
	row := pickRow{stepRow: stepRow{parent: e.Parent, index: e.Index, name: e.Step}}
	at, replaced := -1, 0
	for i, r := range l.rows {
		switch {
		case r.number > 0 && r.name == e.Parent:
			row.side, row.stage, at = r.side, r.name, i+1
		case r.stage != "" && r.parent == e.Parent:
			row.side, row.stage, at = r.side, r.stage, i+1
		case r.stage != "" && r.index == unreadIndex && r.name == e.Parent:
			row.side, row.stage, at, replaced = r.side, r.stage, i, 1
		}
	}
	if at < 0 {
		return l
	}
	l.rows = slices.Replace(slices.Clone(l.rows), at, at+replaced, row)
	return l
}
