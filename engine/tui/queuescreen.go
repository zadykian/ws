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

// sides returns the sides that have a stage, in the order the stages first name them.
func (q queue) sides() []string {
	var sides []string
	for _, s := range q.stages {
		if !slices.Contains(sides, s.side) {
			sides = append(sides, s.side)
		}
	}
	return sides
}

// pane is the place of side's list among the panes.
func (q queue) pane(side string) int { return max(slices.Index(q.sides(), side), 0) }

// panes returns the lists' panes, empty. A side's says so while a role of it stands unread, as a
// check of one machine can't reach what another reports until that one is there.
func (q queue) panes() []Pane {
	var panes []Pane
	for _, side := range q.sides() {
		pane := Pane{Title: side, Focused: q.side == side}
		unread := func(s queueStage) bool {
			return s.side == side && slices.ContainsFunc(s.list.rows, func(r pickRow) bool { return r.index == unreadIndex })
		}
		if slices.ContainsFunc(q.stages, unread) {
			pane.Title += ": roles not scanned"
		}
		panes = append(panes, pane)
	}
	return panes
}

// drawn returns the layout with the lists in its panes, a cursor a side. The side out of focus shows
// none.
func (q queue) drawn(layout Layout, theme Theme, window Size, at map[string]cursor) Layout {
	sizes := layout.PaneSizes(window.Width, window.Height)
	for i, side := range q.sides() {
		there := at[side].within(len(q.lines(side)))
		layout.Panes[i].Body = q.view(theme, side, sizes[i].Height, there, side == q.side)
	}
	return layout
}
