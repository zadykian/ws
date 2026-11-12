package tui

import (
	tea "charm.land/bubbletea/v2"
)

// The places of the queue's buttons that the flat list lacks, after the places that both share.
const (
	queueStatePlace = pickQuitPlace + 1 + iota
	queueConfigurePlace
)

// queueScreen is the queue's lists, the stages' two sides, over the strip of the detail. A scan fills
// them behind the screen, and the maintainer can read and move as it goes.
type queueScreen struct {
	listScreen
	list   pickList
	footer Footer
	// unread is why a machine reported nothing, and its steps stand as the queue named them.
	unread string
	// picks takes the choice, and holds one, so that sending it waits for nobody.
	picks chan<- QueueChoice
}

// Init starts nothing, as the scan is the work's.
func (s queueScreen) Init() tea.Cmd {
	return nil
}

// statusLine is the title's right side, of the stages alone.
func (s queueScreen) statusLine() string {
	on, scanned, all := s.list.tally()
	return s.status(on, scanned, all, len(s.list.passedOver()) > 0)
}

// notes is the words of the strip: what holds for the whole queue, then the line under the cursor.
func (s queueScreen) notes() []Line {
	notes := s.warnings(s.unread, s.list.passedOver())
	line, ok := s.list.current()
	if !ok || line.heading {
		return notes
	}
	row := s.list.rows[line.row]
	if s.list.partial()[row.name] != nil && row.number > 0 {
		notes = append(notes, Line{Said, "Runs in part, so it stays pending."})
	}
	return append(notes, row.notes()...)
}

// layout is the screen with its panes empty, as what fills them depends on their sizes.
func (s queueScreen) layout() Layout {
	detail := Pane{Title: "Detail", Body: s.detail(inside(stripSize(s.window)), s.notes())}
	return Layout{
		Title:  s.title,
		Status: s.statusLine(),
		Panes:  s.list.panes(),
		Below:  strip(s.theme, detail, s.window),
		Footer: s.footer,
	}
}

// listHeight is the lines of the pane of the list in focus that the list has.
func (s queueScreen) listHeight() int {
	height := s.layout().PaneSizes(s.window.Width, s.window.Height)[s.list.pane(s.list.side)].Height
	if s.filtering() {
		height--
	}
	return max(height, 0)
}

// View draws the lists, the filter's line over the one in focus.
func (s queueScreen) View() tea.View {
	l := s.list.drawn(s.layout(), s.theme, s.window)
	if s.filtering() {
		l.Panes[s.list.pane(s.list.side)].Body = s.filter.line(s.theme) + "\n" + s.list.view(s.theme, s.listHeight())
	}
	return l.View(s.theme, s.window.Width, s.window.Height)
}

// typed gives the filter a key or a paste, and the list narrows as its words change.
func (s queueScreen) typed(msg tea.Msg) queueScreen {
	var words string
	var changed bool
	if s.listScreen, words, changed = s.listScreen.typed(msg, s.list.words); changed {
		s.list = s.list.filtered(words)
	}
	return s
}

// queueChoice is what the lists hold, to leave for next.
func (s queueScreen) queueChoice(next Next) QueueChoice {
	c := QueueChoice{Next: next, Side: s.list.side}
	if next == NextRun || next == NextCheck {
		c.Only, c.Partial, c.Whole = s.list.only(), s.list.partial(), s.list.ticked()
	}
	return c
}

// leave gives c to the work. A session quits with its work alone, and a screen never does.
func (s queueScreen) leave(c QueueChoice) (queueScreen, tea.Cmd) {
	s.left = true
	return s, func() tea.Msg {
		s.picks <- c
		return nil
	}
}
