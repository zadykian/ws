package tui

import (
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
)

const (
	// UnreadIndex is a role's in place of a step's, as no event of a run names a role.
	UnreadIndex = -1
	// roleColumns is what a role's heading adds to a step of the role.
	roleColumns = 2
)

// groupForm is how a list shows a group of its steps.
type groupForm int

// The forms of a group: under a heading, bare, and unread.
const (
	GroupHeaded groupForm = iota
	// GroupBare has no heading, as its stage has the name. A filter that misses it brings it back.
	GroupBare
	GroupUnread // a group no scan read: one row by its name
)

// ListGroup is a group with the form it takes in a list.
type ListGroup struct {
	Group
	Form groupForm
}

// NewFormedList returns the list of groups, each step open and left out.
func NewFormedList(groups []ListGroup) PickList {
	var l PickList
	for _, g := range groups {
		if g.Form == GroupUnread {
			l.Rows = append(l.Rows, PickRow{StepRow: StepRow{Index: UnreadIndex, Name: g.Name}, Form: g.Form})
			continue
		}
		for i, name := range g.Steps {
			row := StepRow{Parent: g.Name, Index: i, Name: name}
			l.Rows = append(l.Rows, PickRow{StepRow: row, Form: g.Form, WaitsOn: g.Waits[name]})
		}
	}
	return l
}

// indented is the room of a heading above the row.
func (r PickRow) indented() string {
	if r.Form == GroupHeaded && r.Parent != "" {
		return strings.Repeat(" ", roleColumns)
	}
	return ""
}

// Holds says that the list has the group named parent, by its steps or by its unread row.
func (l PickList) Holds(parent string) bool {
	return slices.ContainsFunc(l.Rows, func(r PickRow) bool {
		return parent != "" && r.Parent == parent || r.Form == GroupUnread && r.Name == parent
	})
}

// Joined returns the list with a row for e's step, where a run has it and no scan named it. A new
// Mac has no mac.conf to name a stage's steps by, and no role of ws has steps before ws is there.
// The step goes after its group's last, in place of the group's unread row. A group the list lacks
// is the stage's own, which has no heading and comes first.
func (l PickList) Joined(e step.Event) PickList {
	if e.Parent == "" || l.Place(e) >= 0 {
		return l
	}
	row := PickRow{StepRow: StepRow{Parent: e.Parent, Index: e.Index, Name: e.Step}, Form: GroupBare}
	at, replaced := 0, 0
	for i, r := range l.Rows {
		switch {
		case r.Parent == e.Parent:
			row.Form, at = r.Form, i+1
		case r.Form == GroupUnread && r.Name == e.Parent:
			row.Form, at, replaced = GroupHeaded, i, 1
		}
	}
	l.Rows = slices.Replace(slices.Clone(l.Rows), at, at+replaced, row)
	return l
}

// Held returns the list with every row locked by those steps that wait on its stage. The stage runs
// whole then, as a partial run would leave it pending, and the stage that waits on it would only skip.
func (l PickList) Held(by []string) PickList {
	l.Rows = slices.Clone(l.Rows)
	for i := range l.Rows {
		l.Rows[i].LockedBy = by
	}
	return l
}

// Boxed returns the list with the box of every row set, the filter's words aside.
func (l PickList) Boxed(on bool) PickList {
	l.Rows = slices.Clone(l.Rows)
	for i := range l.Rows {
		l.Rows[i].Chosen = on
	}
	return l
}
