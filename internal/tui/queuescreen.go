package tui

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
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

// newQueueScreen returns the two lists of stages, their scan under way.
func newQueueScreen(title string, theme Theme, stages []Stage, picks chan<- Choice) pickScreen {
	s := newPickScreen(title, theme, nil, picks)
	s.list, s.queued = newQueue(stages), true
	s.footer = NewFooter(append(slices.Clip(s.footer.buttons), configureButton, stateButton)...)
	return s.enabled()
}

// tabbed walks the ring of the two lists, then the buttons, as a Tab between the lists would else
// leave for the buttons.
func (s pickScreen) tabbed(step int) pickScreen {
	next := int(s.list.side) + step
	if s.footer.Focused() < 0 && next >= 0 && next < int(sideCount) {
		s.list = s.list.facing(Side(next))
		return s
	}
	s.footer = s.footer.move(step)
	if s.footer.Focused() >= 0 {
		return s
	}
	s.list = s.list.facing(MacSide)
	if step < 0 {
		s.list = s.list.facing(WsSide)
	}
	return s
}

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

// panes returns the two lists' panes, empty. ws's says so while a role stands unread, as a check of
// the Mac can't reach what ws reports until ws is there.
func (l pickList) panes() []Pane {
	ws := "ws"
	if slices.ContainsFunc(l.rows, func(r pickRow) bool { return r.index == unreadIndex }) {
		ws = "ws: roles not scanned"
	}
	return []Pane{{Title: "Mac", Focused: l.side == MacSide}, {Title: ws, Focused: l.side == WsSide}}
}

// drawn returns the layout with the two lists in its panes.
func (l pickList) drawn(layout Layout, theme Theme, window Size) Layout {
	sizes := layout.PaneSizes(window.Width, window.Height)
	for side := range sideCount {
		layout.Panes[side].Body = l.glance(side).view(theme, sizes[side].Height)
	}
	return layout
}

// queueLayout is the screen of a queue, its lists empty as what fills them depends on their sizes.
func (s pickScreen) queueLayout() Layout {
	detail := Pane{Title: "Detail", Body: s.detail(inside(stripSize(s.window)))}
	return Layout{
		Title:  s.title,
		Status: s.status(),
		Panes:  s.list.panes(),
		Below:  strip(s.theme, detail, s.window),
		Footer: s.footer,
	}
}

// queueView draws the two lists, the filter's line over the one in focus.
func (s pickScreen) queueView() tea.View {
	l := s.list.drawn(s.layout(), s.theme, s.window)
	if s.filtering() {
		l.Panes[s.list.side].Body = s.filterLine() + "\n" + s.list.view(s.theme, s.listHeight())
	}
	return l.View(s.theme, s.window.Width, s.window.Height)
}
