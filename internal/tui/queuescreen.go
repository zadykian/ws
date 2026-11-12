package tui

import (
	"slices"

	"charm.land/bubbles/v2/key"
)

const (
	// stripWindow is the lines under which a window leaves the strip out, for the lists to read.
	stripWindow = 14
	// stripLeast and stripMost bound the strip, frame included, around a third of the window.
	stripLeast = 5
	stripMost  = 12
)

// configureButton leaves the lists for the values' form, which a scan needs no end for.
var configureButton = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "configure"))

// stateButton leaves the lists for the state screen of the machine of the list in focus.
var stateButton = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "state"))

// stripSize is the frame of the strip under a queue's lists, and the zero Size where a window has
// no room for one.
func stripSize(window Size) Size {
	if window.Height < stripWindow || window.Width < minFrameWidth {
		return Size{}
	}
	return Size{window.Width, min(max(window.Height/3, stripLeast), stripMost)}
}

// strip draws pane in the strip of a window, or nothing where the window has no room.
func strip(theme Theme, pane Pane, window Size) string {
	size := stripSize(window)
	if size == (Size{}) {
		return ""
	}
	return pane.render(theme, size)
}

// inside is the room within a frame of that size.
func inside(frame Size) Size {
	return Size{max(frame.Width-2, 0), max(frame.Height-2, 0)}
}

// sides returns the sides that have a list: ws's alone where no stage runs on the Mac, as in ws
// server wizard, else both.
func (l pickList) sides() []Side {
	if slices.ContainsFunc(l.rows, func(r pickRow) bool { return r.side == MacSide }) {
		return []Side{MacSide, WsSide}
	}
	return []Side{WsSide}
}

// pane is the place of side's list among the panes.
func (l pickList) pane(side Side) int { return max(slices.Index(l.sides(), side), 0) }

// panes returns the lists' panes, empty. ws's says so while a role stands unread, as a check of the
// Mac can't reach what ws reports until ws is there.
func (l pickList) panes() []Pane {
	ws := Pane{Title: WsSide.String(), Focused: l.side == WsSide}
	if slices.ContainsFunc(l.rows, func(r pickRow) bool { return r.index == unreadIndex }) {
		ws.Title = WsSide.String() + ": roles not scanned"
	}
	if len(l.sides()) == 1 {
		return []Pane{ws}
	}
	return []Pane{{Title: MacSide.String(), Focused: l.side == MacSide}, ws}
}

// drawn returns the layout with the lists in its panes, a cursor a side. The side out of focus shows
// none.
func (l pickList) drawn(layout Layout, theme Theme, window Size, at [2]cursor) Layout {
	sizes := layout.PaneSizes(window.Width, window.Height)
	for i, side := range l.sides() {
		seen := l
		seen.side = side
		there := at[side].within(len(seen.lines()))
		layout.Panes[i].Body = seen.view(theme, sizes[i].Height, there, side == l.side)
	}
	return layout
}

// sides returns the sides that have a stage: ws's alone where none runs on the Mac, as in ws server
// wizard, else both.
func (q queue) sides() []Side {
	if slices.ContainsFunc(q.stages, func(s queueStage) bool { return s.side == MacSide }) {
		return []Side{MacSide, WsSide}
	}
	return []Side{WsSide}
}

// pane is the place of side's list among the panes.
func (q queue) pane(side Side) int { return max(slices.Index(q.sides(), side), 0) }

// panes returns the lists' panes, empty, the one of the side in focus focused. ws's says so while a
// role stands unread, as a check of the Mac can't reach what ws reports until ws is there.
func (q queue) panes(focus Side) []Pane {
	ws := Pane{Title: WsSide.String(), Focused: focus == WsSide}
	unread := func(s queueStage) bool {
		return slices.ContainsFunc(s.list.rows, func(r pickRow) bool { return r.index == unreadIndex })
	}
	if slices.ContainsFunc(q.stages, unread) {
		ws.Title = WsSide.String() + ": roles not scanned"
	}
	if len(q.sides()) == 1 {
		return []Pane{ws}
	}
	return []Pane{{Title: MacSide.String(), Focused: focus == MacSide}, ws}
}

// drawn returns the layout with the sides' lines in its panes, a cursor a side. The side out of
// focus shows none.
func (q queue) drawn(layout Layout, theme Theme, window Size, focus Side, at [2]cursor) Layout {
	sizes := layout.PaneSizes(window.Width, window.Height)
	for i, side := range q.sides() {
		there := at[side].within(len(q.lines(side)))
		layout.Panes[i].Body = q.view(theme, side, sizes[i].Height, there, side == focus)
	}
	return layout
}
