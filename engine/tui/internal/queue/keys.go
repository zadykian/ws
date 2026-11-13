package queue

import (
	"maps"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/run"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The places of the queue's buttons that the flat list lacks, after the places that both share.
const (
	queueStatePlace = list.PickQuitPlace + 1 + iota
	queueConfigurePlace
)

// queueScreen is the queue's lists, a side each, over the strip of the detail. A scan fills them
// behind the screen, and the maintainer can read and move as it goes.
type queueScreen struct {
	list.Screen
	list queue
	// at is the cursor on each side's list, by side. Never written in place: the screens are values,
	// and Reopen reuses a left one, so withCursor copies it.
	at     map[string]widget.Cursor
	footer widget.Footer
	// unread is why a machine reported nothing, and its steps stand as the queue named them.
	unread string
	// picks takes the choice, and holds one, so that sending it waits for nobody.
	picks chan<- Choice
}

// openQueue returns the lists of stages, their scan under way. The configure button comes last.
func openQueue(title string, theme widget.Theme, stages []Stage, picks chan<- Choice) queueScreen {
	buttons := append(list.Buttons(), stateButton, configureButton)
	s := queueScreen{Screen: list.Screen{Title: title, Theme: theme, Scanning: true, Filter: widget.NewFilter()},
		list: queueOf(stages), footer: widget.NewFooter(buttons...), picks: picks}
	return s.enabled()
}

// Init starts nothing, as the scan is the work's.
func (s queueScreen) Init() tea.Cmd {
	return nil
}

// statusLine is the title's right side, of the stages alone.
func (s queueScreen) statusLine() string {
	on, scanned, all := s.list.count()
	return s.Status(on, scanned, all, len(s.list.passed()) > 0)
}

// notes is the words of the strip: what holds for the whole queue, then the line under the cursor.
func (s queueScreen) notes() []widget.Line {
	notes := s.Warnings(s.unread, s.list.passed())
	lines := s.list.lines(s.list.side)
	at := int(s.at[s.list.side])
	if at >= len(lines) || lines[at].line.Heading {
		return notes
	}
	line := lines[at]
	row := s.list.stages[line.stage].row
	switch {
	case !line.head:
		row = s.list.stages[line.stage].list.Rows[line.line.Row]
	case s.list.partial()[row.Name] != nil:
		notes = append(notes, widget.Line{Kind: widget.Said, Text: "Runs in part, so it stays pending."})
	}
	return append(notes, row.Notes()...)
}

// layout is the screen with its panes empty, as what fills them depends on their sizes.
func (s queueScreen) layout() widget.Layout {
	detail := widget.Pane{Title: "Detail", Body: s.Detail(inside(stripSize(s.Window)), s.notes())}
	return widget.Layout{
		Title:  s.Title,
		Status: s.statusLine(),
		Panes:  s.list.panes(),
		Below:  strip(s.Theme, detail, s.Window),
		Footer: s.footer,
	}
}

// listHeight is the lines of the pane of the list in focus that the list has.
func (s queueScreen) listHeight() int {
	sizes := s.layout().PaneSizes(s.Window.Width, s.Window.Height)
	at := s.list.pane(s.list.side)
	if at >= len(sizes) {
		return 0
	}
	height := sizes[at].Height
	if s.Filtering() {
		height--
	}
	return max(height, 0)
}

// View draws the lists, the filter's line over the one in focus.
func (s queueScreen) View() tea.View {
	l := s.list.drawn(s.layout(), s.Theme, s.Window, s.at)
	if at := s.list.pane(s.list.side); s.Filtering() && at < len(l.Panes) {
		l.Panes[at].Body = s.Filter.Line(s.Theme) + "\n" + s.list.view(s.Theme, s.list.side, s.listHeight(), s.at[s.list.side], true)
	}
	return l.View(s.Theme, s.Window.Width, s.Window.Height)
}

// Update takes the scan's events and lines, the window's size and the keys.
func (s queueScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case step.Event:
		s.list = s.list.take(msg)
	case widget.Line:
		s.list = s.list.hear(msg)
	case list.ScannedMsg:
		s.Screen = s.Scanned(msg)
		s.list = s.list.opened()
	case run.ReportedMsg:
		s.unread = msg.Problem
		if msg.Problem == "" {
			s.list = s.list.reported(msg.Stage, msg.Sections)
		}
	case tea.WindowSizeMsg:
		s.Window = widget.Size{Width: msg.Width, Height: msg.Height}
	case tea.PasteMsg:
		if s.Filter.Editing && !s.Left {
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
	case s.Left:
		return s, nil
	case s.Held():
		return s.typed(msg), nil
	case key.Matches(msg, widget.ClearKey):
		s.Filter = s.Filter.Cleared()
		s = s.narrowed("")
		return s, nil
	case key.Matches(msg, widget.Keys.Next):
		return s.tabbed(1), nil
	case key.Matches(msg, widget.Keys.Prev):
		return s.tabbed(-1), nil
	}
	var place int
	s.footer, place = s.footer.Update(msg)
	switch place {
	case list.PickTogglePlace:
		s.list = s.list.toggled(s.list.side, s.at[s.list.side])
	case list.PickAllPlace:
		s.list = s.list.chose(func(list.PickRow) bool { return true })
	case list.PickNonePlace:
		s.list = s.list.chose(func(list.PickRow) bool { return false })
	case list.PickInvertPlace:
		s.list = s.list.chose(func(r list.PickRow) bool { return !r.On() })
	case list.PickPendingPlace:
		s.list = s.list.chose(list.PickRow.Pending).whole(func(o queueStage) bool { return o.row.Pending() })
	case list.PickFilterPlace:
		s.Filter = s.Filter.Opened()
	case list.PickRunPlace:
		return s.leave(s.queueChoice(NextRun))
	case list.PickCheckPlace:
		return s.leave(s.queueChoice(NextCheck))
	case list.PickQuitPlace:
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
	if s.Screen, words, changed = s.Typed(msg, s.list.words); changed {
		s = s.narrowed(words)
	}
	return s
}

// narrowed returns s with its lists under words, and the cursor of the side in focus kept.
func (s queueScreen) narrowed(words string) queueScreen {
	var c widget.Cursor
	s.list, c = s.list.narrowed(s.list.side, s.at[s.list.side], words)
	s.at = withCursor(s.at, s.list.side, c)
	return s
}

// withCursor returns at with side's cursor set, on a copy.
func withCursor(at map[string]widget.Cursor, side string, c widget.Cursor) map[string]widget.Cursor {
	next := maps.Clone(at)
	if next == nil {
		next = map[string]widget.Cursor{}
	}
	next[side] = c
	return next
}

// queueChoice is what the lists hold, to leave for next.
func (s queueScreen) queueChoice(next Next) Choice {
	c := Choice{Next: next, Side: s.list.side, left: s}
	if next == NextRun || next == NextCheck {
		c.Only, c.Partial, c.Whole = s.list.only(), s.list.partial(), s.list.ticked()
	}
	return c
}

// leave gives c to the work. A session quits with its work alone, and a screen never does.
func (s queueScreen) leave(c Choice) (queueScreen, tea.Cmd) {
	s.Left = true
	return s, func() tea.Msg {
		s.picks <- c
		return nil
	}
}

// enabled returns s with its boxes held while the scan ticks them, as a press would race it. A run
// waits for something chosen, as a run with no Only runs every step. The state screen waits for the
// scan's end, as both read the same machine.
func (s queueScreen) enabled() queueScreen {
	for place := list.PickTogglePlace; place <= list.PickPendingPlace; place++ {
		s.footer = s.footer.Enable(place, !s.Scanning)
	}
	some := !s.Scanning && len(s.list.only()) > 0
	s.footer = s.footer.Enable(list.PickRunPlace, some).Enable(list.PickCheckPlace, some)
	s.footer = s.footer.Enable(queueStatePlace, !s.Scanning)
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
