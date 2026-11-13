package tui

import (
	"maps"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/step"
)

// The places of the queue's buttons that the flat list lacks, after the places that both share.
const (
	queueStatePlace = pickQuitPlace + 1 + iota
	queueConfigurePlace
)

// reportedMsg says what a stage's machine reports, or why it reports nothing.
type reportedMsg struct {
	stage    string
	sections []StateSection
	problem  string
}

// queueScreen is the queue's lists, a side each, over the strip of the detail. A scan fills them
// behind the screen, and the maintainer can read and move as it goes.
type queueScreen struct {
	listScreen
	list queue
	// at is the cursor on each side's list, by side. Never written in place: the screens are values,
	// and Reopen reuses a left one, so withCursor copies it.
	at     map[string]Cursor
	footer Footer
	// unread is why a machine reported nothing, and its steps stand as the queue named them.
	unread string
	// picks takes the choice, and holds one, so that sending it waits for nobody.
	picks chan<- QueueChoice
}

// openQueue returns the lists of stages, their scan under way. The configure button comes last.
func openQueue(title string, theme Theme, stages []Stage, picks chan<- QueueChoice) queueScreen {
	buttons := append(listButtons(), stateButton, configureButton)
	s := queueScreen{listScreen: listScreen{title: title, theme: theme, scanning: true, filter: NewFilter()},
		list: queueOf(stages), footer: NewFooter(buttons...), picks: picks}
	return s.enabled()
}

// Init starts nothing, as the scan is the work's.
func (s queueScreen) Init() tea.Cmd {
	return nil
}

// statusLine is the title's right side, of the stages alone.
func (s queueScreen) statusLine() string {
	on, scanned, all := s.list.count()
	return s.status(on, scanned, all, len(s.list.passed()) > 0)
}

// notes is the words of the strip: what holds for the whole queue, then the line under the cursor.
func (s queueScreen) notes() []Line {
	notes := s.warnings(s.unread, s.list.passed())
	lines := s.list.lines(s.list.side)
	at := int(s.at[s.list.side])
	if at >= len(lines) || lines[at].line.heading {
		return notes
	}
	line := lines[at]
	row := s.list.stages[line.stage].row
	switch {
	case !line.head:
		row = s.list.stages[line.stage].list.rows[line.line.row]
	case s.list.partial()[row.name] != nil:
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
	l := s.list.drawn(s.layout(), s.theme, s.window, s.at)
	if s.filtering() {
		l.Panes[s.list.pane(s.list.side)].Body = s.filter.Line(s.theme) + "\n" + s.list.view(s.theme, s.list.side, s.listHeight(), s.at[s.list.side], true)
	}
	return l.View(s.theme, s.window.Width, s.window.Height)
}

// Update takes the scan's events and lines, the window's size and the keys.
func (s queueScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case step.Event:
		s.list = s.list.take(msg)
	case Line:
		s.list = s.list.hear(msg)
	case scannedMsg:
		s.listScreen = s.scanned(msg)
		s.list = s.list.opened()
	case reportedMsg:
		s.unread = msg.problem
		if msg.problem == "" {
			s.list = s.list.reported(msg.stage, msg.sections)
		}
	case tea.WindowSizeMsg:
		s.window = Size{msg.Width, msg.Height}
	case tea.PasteMsg:
		if s.filter.Editing && !s.left {
			s = s.typed(msg)
		}
	case tea.KeyPressMsg:
		var leaving tea.Cmd
		s, leaving = s.pressed(msg)
		return s.enabled(), leaving
	}
	return s.enabled(), nil
}

// pressed turns the boxes that msg's button names, moves the cursor, walks the lists, or gives the
// choice.
func (s queueScreen) pressed(msg tea.KeyPressMsg) (queueScreen, tea.Cmd) {
	switch {
	case s.left:
		return s, nil
	case s.held():
		return s.typed(msg), nil
	case key.Matches(msg, ClearKey):
		s.filter = s.filter.Cleared()
		s = s.narrowed("")
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
		s.list = s.list.toggled(s.list.side, s.at[s.list.side])
	case pickAllPlace:
		s.list = s.list.chose(func(pickRow) bool { return true })
	case pickNonePlace:
		s.list = s.list.chose(func(pickRow) bool { return false })
	case pickInvertPlace:
		s.list = s.list.chose(func(r pickRow) bool { return !r.on() })
	case pickPendingPlace:
		s.list = s.list.chose(pickRow.pending).whole(func(o queueStage) bool { return o.row.pending() })
	case pickFilterPlace:
		s.filter = s.filter.Opened()
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
		s.at = withCursor(s.at, s.list.side, s.at[s.list.side].Moved(msg, max(s.listHeight(), 1), len(s.list.lines(s.list.side))))
	}
	return s, nil
}

// typed gives the filter a key or a paste, and the list narrows as its words change.
func (s queueScreen) typed(msg tea.Msg) queueScreen {
	var words string
	var changed bool
	if s.listScreen, words, changed = s.listScreen.typed(msg, s.list.words); changed {
		s = s.narrowed(words)
	}
	return s
}

// narrowed returns s with its lists under words, and the cursor of the side in focus kept.
func (s queueScreen) narrowed(words string) queueScreen {
	var c Cursor
	s.list, c = s.list.narrowed(s.list.side, s.at[s.list.side], words)
	s.at = withCursor(s.at, s.list.side, c)
	return s
}

// withCursor returns at with side's cursor set, on a copy.
func withCursor(at map[string]Cursor, side string, c Cursor) map[string]Cursor {
	next := maps.Clone(at)
	if next == nil {
		next = map[string]Cursor{}
	}
	next[side] = c
	return next
}

// queueChoice is what the lists hold, to leave for next.
func (s queueScreen) queueChoice(next Next) QueueChoice {
	c := QueueChoice{Next: next, Side: s.list.side, left: s}
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

// enabled returns s with its boxes held while the scan ticks them, as a press would race it. A run
// waits for something chosen, as a run with no Only runs every step. The state screen waits for the
// scan's end, as both read the same machine.
func (s queueScreen) enabled() queueScreen {
	for place := pickTogglePlace; place <= pickPendingPlace; place++ {
		s.footer = s.footer.Enable(place, !s.scanning)
	}
	some := !s.scanning && len(s.list.only()) > 0
	s.footer = s.footer.Enable(pickRunPlace, some).Enable(pickCheckPlace, some)
	s.footer = s.footer.Enable(queueStatePlace, !s.scanning)
	return s
}

// tabbed walks the ring of the lists, then the buttons, as a Tab between the lists would else leave
// for the buttons.
func (s queueScreen) tabbed(step int) queueScreen {
	sides := s.list.sides()
	next := s.list.pane(s.list.side) + step
	if s.footer.Focused() < 0 && next >= 0 && next < len(sides) {
		s = s.faced(sides[next])
		return s
	}
	s.footer = s.footer.Move(step)
	if s.footer.Focused() >= 0 {
		return s
	}
	s = s.faced(sides[0])
	if step < 0 {
		s = s.faced(sides[len(sides)-1])
	}
	return s
}

// faced returns s turned to side, whose cursor a filter can have left past its list.
func (s queueScreen) faced(side string) queueScreen {
	s.list.side = side
	s.at = withCursor(s.at, side, s.at[side].Within(len(s.list.lines(side))))
	return s
}
