package tui

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
	if r.form == groupHeaded {
		return r.indent()
	}
	return ""
}
