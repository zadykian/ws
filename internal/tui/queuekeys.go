package tui

import (
	"charm.land/bubbles/v2/key"
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

// pressed turns the boxes that msg's button names, moves the cursor, walks the lists, or gives the
// choice.
func (s queueScreen) pressed(msg tea.KeyPressMsg) (queueScreen, tea.Cmd) {
	switch {
	case s.left:
		return s, nil
	case s.held():
		return s.typed(msg), nil
	case key.Matches(msg, clearKey):
		s.filter = s.filter.cleared()
		s.list = s.list.filtered("")
		return s, nil
	case key.Matches(msg, Keys.Next):
		return s.tabbed(1), nil
	case key.Matches(msg, Keys.Prev):
		return s.tabbed(-1), nil
	}
	var place int
	s.footer, place = s.footer.Update(msg)
	switch place {
	case pickTogglePlace:
		s.list = s.list.toggled()
	case pickAllPlace:
		s.list = s.list.chose(func(pickRow) bool { return true })
	case pickNonePlace:
		s.list = s.list.chose(func(pickRow) bool { return false })
	case pickInvertPlace:
		s.list = s.list.chose(func(r pickRow) bool { return !r.on() })
	case pickPendingPlace:
		s.list = s.list.chose(pickRow.pending).whole(pickRow.pending)
	case pickFilterPlace:
		s.filter = s.filter.opened()
	case pickRunPlace:
		return s.leave(s.queueChoice(NextRun))
	case pickCheckPlace:
		return s.leave(s.queueChoice(NextCheck))
	case pickQuitPlace:
		return s.leave(s.queueChoice(NextLeave))
	case queueConfigurePlace:
		return s.leave(s.queueChoice(NextConfigure))
	case queueStatePlace:
		return s.leave(s.queueChoice(NextState))
	default:
		s.list = s.list.moved(msg, max(s.listHeight(), 1))
	}
	return s, nil
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

// tabbed walks the ring of the lists, then the buttons, as a Tab between the lists would else leave
// for the buttons.
func (s queueScreen) tabbed(step int) queueScreen {
	sides := s.list.sides()
	next := s.list.pane(s.list.side) + step
	if s.footer.Focused() < 0 && next >= 0 && next < len(sides) {
		s.list = s.list.facing(sides[next])
		return s
	}
	s.footer = s.footer.move(step)
	if s.footer.Focused() >= 0 {
		return s
	}
	s.list = s.list.facing(sides[0])
	if step < 0 {
		s.list = s.list.facing(sides[len(sides)-1])
	}
	return s
}
