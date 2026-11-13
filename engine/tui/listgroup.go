package tui

import (
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
)

const (
	// unreadIndex is a role's in place of a step's, as no event of a run names a role.
	unreadIndex = -1
	// roleColumns is what a role's heading adds to a step of the role.
	roleColumns = 2
)

// groupForm is how a list shows a group of its steps.
type groupForm int

// The forms of a group: under a heading, bare, and unread.
const (
	groupHeaded groupForm = iota
	// groupBare has no heading, as its stage has the name. A filter that misses it brings it back.
	groupBare
	groupUnread // a group no scan read: one row by its name
)

// listGroup is a group with the form it takes in a list.
type listGroup struct {
	Group
	form groupForm
}

// newFormedList returns the list of groups, each step open and left out.
func newFormedList(groups []listGroup) pickList {
	var l pickList
	for _, g := range groups {
		if g.form == groupUnread {
			l.rows = append(l.rows, pickRow{stepRow: stepRow{index: unreadIndex, name: g.Name}, form: g.form})
			continue
		}
		for i, name := range g.Steps {
			row := stepRow{parent: g.Name, index: i, name: name}
			l.rows = append(l.rows, pickRow{stepRow: row, form: g.form, waitsOn: g.Waits[name]})
		}
	}
	return l
}

// indented is the room of a heading above the row.
func (r pickRow) indented() string {
	if r.form == groupHeaded && r.parent != "" {
		return strings.Repeat(" ", roleColumns)
	}
	return ""
}

// holds says that the list has the group named parent, by its steps or by its unread row.
func (l pickList) holds(parent string) bool {
	return slices.ContainsFunc(l.rows, func(r pickRow) bool {
		return parent != "" && r.parent == parent || r.form == groupUnread && r.name == parent
	})
}

// joined returns the list with a row for e's step, where a run has it and no scan named it. A new
// Mac has no mac.conf to name a stage's steps by, and no role of ws has steps before ws is there.
// The step goes after its group's last, in place of the group's unread row. A group the list lacks
// is the stage's own, which has no heading and comes first.
func (l pickList) joined(e step.Event) pickList {
	if e.Parent == "" || l.place(e) >= 0 {
		return l
	}
	row := pickRow{stepRow: stepRow{parent: e.Parent, index: e.Index, name: e.Step}, form: groupBare}
	at, replaced := 0, 0
	for i, r := range l.rows {
		switch {
		case r.parent == e.Parent:
			row.form, at = r.form, i+1
		case r.form == groupUnread && r.name == e.Parent:
			row.form, at, replaced = groupHeaded, i, 1
		}
	}
	l.rows = slices.Replace(slices.Clone(l.rows), at, at+replaced, row)
	return l
}

// held returns the list with every row locked by those steps that wait on its stage. The stage runs
// whole then, as a partial run would leave it pending, and the stage that waits on it would only skip.
func (l pickList) held(by []string) pickList {
	l.rows = slices.Clone(l.rows)
	for i := range l.rows {
		l.rows[i].lockedBy = by
	}
	return l
}

// boxed returns the list with the box of every row set, the filter's words aside.
func (l pickList) boxed(on bool) pickList {
	l.rows = slices.Clone(l.rows)
	for i := range l.rows {
		l.rows[i].chosen = on
	}
	return l
}
