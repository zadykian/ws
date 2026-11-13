package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/run"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// Choice is what a flat list leaves to a run. No steps says that nothing runs.
type Choice struct {
	// Only names the steps as Options.Only does.
	Only []string
	// Check plans them and applies nothing.
	Check bool
}

// pickScreen is the list that a run is chosen from. A scan fills it behind the screen, and the
// maintainer can read and move as it goes.
type pickScreen struct {
	list.Screen
	list list.PickList
	// at is the cursor on the list.
	at     widget.Cursor
	footer widget.Footer
	// picks takes the choice, and holds one, so that sending it waits for nobody.
	picks chan<- Choice
}

// newPickScreen returns the screen of groups, its scan under way.
func newPickScreen(title string, theme widget.Theme, groups []list.Group, picks chan<- Choice) pickScreen {
	footer := widget.NewFooter(list.Buttons()...)
	s := pickScreen{Screen: list.Screen{Title: title, Theme: theme, Scanning: true, Filter: widget.NewFilter()},
		list: list.NewPickList(groups), footer: footer, picks: picks}
	return s.enabled()
}

// Init starts nothing, as the scan is the work's.
func (s pickScreen) Init() tea.Cmd {
	return nil
}

// Update takes the scan's events and lines, the window's size and the keys.
func (s pickScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case step.Event:
		s.list = s.list.Take(msg)
	case widget.Line:
		s.list = s.list.Hear(msg)
	case list.ScannedMsg:
		s.Screen = s.Scanned(msg)
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

// pressed turns the boxes that msg's button names, moves the cursor, or gives the choice.
func (s pickScreen) pressed(msg tea.KeyPressMsg) (pickScreen, tea.Cmd) {
	switch {
	case s.Left:
		return s, nil
	case s.Held():
		return s.typed(msg), nil
	case key.Matches(msg, widget.ClearKey):
		s.Filter = s.Filter.Cleared()
		s.list, s.at = s.list.Narrowed(s.at, "")
		return s, nil
	}
	var place int
	s.footer, place = s.footer.Update(msg)
	switch place {
	case list.PickTogglePlace:
		s.list = s.list.Toggled(s.at)
	case list.PickAllPlace:
		s.list = s.list.Chose(func(list.PickRow) bool { return true })
	case list.PickNonePlace:
		s.list = s.list.Chose(func(list.PickRow) bool { return false })
	case list.PickInvertPlace:
		s.list = s.list.Chose(func(r list.PickRow) bool { return !r.On() })
	case list.PickPendingPlace:
		s.list = s.list.Chose(list.PickRow.Pending)
	case list.PickFilterPlace:
		s.Filter = s.Filter.Opened()
	case list.PickRunPlace:
		return s.leave(Choice{Only: s.list.Only()})
	case list.PickCheckPlace:
		return s.leave(Choice{Only: s.list.Only(), Check: true})
	case list.PickQuitPlace:
		return s.leave(Choice{})
	default:
		s.at = s.at.Moved(msg, max(s.listHeight(), 1), len(s.list.Lines()))
	}
	return s, nil
}

// typed gives the filter a key or a paste, and the list narrows as its words change.
func (s pickScreen) typed(msg tea.Msg) pickScreen {
	var words string
	var changed bool
	if s.Screen, words, changed = s.Typed(msg, s.list.Words); changed {
		s.list, s.at = s.list.Narrowed(s.at, words)
	}
	return s
}

// leave gives c to the work. A session quits with its work alone, and a screen never does.
func (s pickScreen) leave(c Choice) (pickScreen, tea.Cmd) {
	s.Left = true
	return s, func() tea.Msg {
		s.picks <- c
		return nil
	}
}

// enabled returns s with its boxes held while the scan ticks them, as a press would race it. A run
// waits for something chosen, as a run with no Only runs every step. The state screen waits for the
// scan's end, as both read the same machine.
func (s pickScreen) enabled() pickScreen {
	for place := list.PickTogglePlace; place <= list.PickPendingPlace; place++ {
		s.footer = s.footer.Enable(place, !s.Scanning)
	}
	some := !s.Scanning && len(s.list.Only()) > 0
	s.footer = s.footer.Enable(list.PickRunPlace, some).Enable(list.PickCheckPlace, some)
	return s
}

// statusLine is the title's right side, of the steps alone, a queue's stages leaving out.
func (s pickScreen) statusLine() string {
	on, scanned, all := s.list.Tally()
	return s.Status(on, scanned, all, len(s.list.PassedOver()) > 0)
}

// layout is the screen with its panes empty, as what fills them depends on their sizes. The list
// holds the focus alone, so that a window too short for two panes shows it.
func (s pickScreen) layout() widget.Layout {
	return widget.Layout{
		Title:  s.Title,
		Status: s.statusLine(),
		Panes:  []widget.Pane{{Title: "Steps", Focused: true}, {Title: "Detail"}},
		Footer: s.footer,
	}
}

// listHeight is the lines of the pane of steps that the list has.
func (s pickScreen) listHeight() int {
	height := s.layout().PaneSizes(s.Window.Width, s.Window.Height)[0].Height
	if s.Filtering() {
		height--
	}
	return max(height, 0)
}

// View draws the screen.
func (s pickScreen) View() tea.View {
	l := s.layout()
	l.Panes[1].Body = s.Detail(l.PaneSizes(s.Window.Width, s.Window.Height)[1], s.notes())
	l.Panes[0].Body = s.list.View(s.Theme, s.listHeight(), s.at)
	if s.Filtering() {
		l.Panes[0].Body = s.Filter.Line(s.Theme) + "\n" + l.Panes[0].Body
	}
	return l.View(s.Theme, s.Window.Width, s.Window.Height)
}

// notes is the words of the pane: what holds for the whole list, then the line under the cursor.
func (s pickScreen) notes() []widget.Line {
	notes := s.Warnings("", s.list.PassedOver())
	line, ok := s.list.Current(s.at)
	if !ok || line.Heading {
		return notes
	}
	row := s.list.Rows[line.Row]
	return append(notes, row.Notes()...)
}

// Pick draws the list of groups over scan, a run with --check, and returns what the maintainer
// chose: a Choice with no steps where they left. Ctrl-C ends it as Cut says, and no stop is asked.
func Pick(ctx context.Context, c Config, title string, groups []list.Group, scan Work) (Choice, error) {
	picks := make(chan Choice, 1)
	return run.Await(ctx, c, newPickScreen(title, c.Theme, groups, picks), picks, scan)
}
