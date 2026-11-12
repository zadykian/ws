package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/step"
)

// The places of the list's buttons.
const (
	pickTogglePlace = iota
	pickAllPlace
	pickNonePlace
	pickInvertPlace
	pickPendingPlace
	pickFilterPlace
	pickRunPlace
	pickCheckPlace
	pickQuitPlace
)

// Choice is what a flat list leaves to a run. No steps says that nothing runs.
type Choice struct {
	// Only names the steps as Options.Only does.
	Only []string
	// Check plans them and applies nothing.
	Check bool
}

// scannedMsg ends the scan. problem says why it stopped short, if it did.
type scannedMsg struct{ problem string }

// listScreen is what both screens of lists share: the frame, the scan's end, the filter's typing
// and the leave.
type listScreen struct {
	title  string
	theme  Theme
	window Size
	// scanning says that events of the scan are still on their way.
	scanning bool
	// problem is why the scan stopped short.
	problem string
	// filter holds the words that the list follows.
	filter filter
	// left says that the choice has gone, and the screen waits for the work to end it.
	left bool
}

// scanned ends the scan, and keeps why it stopped short, if it did.
func (s listScreen) scanned(msg scannedMsg) listScreen {
	s.scanning, s.problem = false, msg.problem
	return s
}

// typed gives the filter a key or a paste, and returns its words and whether they differ from was.
func (s listScreen) typed(msg tea.Msg, was string) (listScreen, string, bool) {
	s.filter = s.filter.edited(msg)
	words := s.filter.words()
	return s, words, words != was
}

// held says that no button takes a key: the screen is leaving, or the filter takes it.
func (s listScreen) held() bool { return s.left || s.filter.editing }

// status is the title's right side: the scan's progress, then what is chosen. Its mark stands for
// the warning of a role passed over, which a short window has no pane for.
func (s listScreen) status(on, scanned, all int, warned bool) string {
	switch {
	case s.left:
		return "leaving"
	case s.scanning:
		return fmt.Sprintf("scanning %d/%d", scanned, all)
	case warned:
		return fmt.Sprintf("! %d of %d chosen", on, all)
	}
	return fmt.Sprintf("%d of %d chosen", on, all)
}

// filtering says that the filter's line shows, over the list.
func (s listScreen) filtering() bool { return s.filter.shows() }

// pickScreen is the list that a run is chosen from. A scan fills it behind the screen, and the
// maintainer can read and move as it goes.
type pickScreen struct {
	listScreen
	list   pickList
	footer Footer
	// picks takes the choice, and holds one, so that sending it waits for nobody.
	picks chan<- Choice
}

// listButtons are the buttons of both screens of lists, in the order of their places.
func listButtons() []key.Binding {
	button := func(k, label string) key.Binding { return key.NewBinding(key.WithKeys(k), key.WithHelp(k, label)) }
	return []key.Binding{button("space", "toggle"), button("a", "all"), button("n", "none"), button("i", "invert"),
		button("p", "pending"), button("/", "filter"), button("enter", "run"), button("c", "check"), Keys.Quit}
}

// newPickScreen returns the screen of groups, its scan under way.
func newPickScreen(title string, theme Theme, groups []Group, picks chan<- Choice) pickScreen {
	footer := NewFooter(listButtons()...)
	s := pickScreen{listScreen: listScreen{title: title, theme: theme, scanning: true, filter: newFilter()},
		list: newPickList(groups), footer: footer, picks: picks}
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
		s.list = s.list.take(msg)
	case Line:
		s.list = s.list.hear(msg)
	case scannedMsg:
		s.listScreen = s.scanned(msg)
	case tea.WindowSizeMsg:
		s.window = Size{msg.Width, msg.Height}
	case tea.PasteMsg:
		if s.filter.editing && !s.left {
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
	case s.left:
		return s, nil
	case s.held():
		return s.typed(msg), nil
	case key.Matches(msg, clearKey):
		s.filter = s.filter.cleared()
		s.list, s.list.at = s.list.narrowed(s.list.at, "")
		return s, nil
	}
	var place int
	s.footer, place = s.footer.Update(msg)
	switch place {
	case pickTogglePlace:
		s.list = s.list.toggled(s.list.at)
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
		return s.leave(Choice{Only: s.list.only()})
	case pickCheckPlace:
		return s.leave(Choice{Only: s.list.only(), Check: true})
	case pickQuitPlace:
		return s.leave(Choice{})
	default:
		s.list.at = s.list.at.moved(msg, max(s.listHeight(), 1), len(s.list.lines()))
	}
	return s, nil
}

// typed gives the filter a key or a paste, and the list narrows as its words change.
func (s pickScreen) typed(msg tea.Msg) pickScreen {
	var words string
	var changed bool
	if s.listScreen, words, changed = s.listScreen.typed(msg, s.list.words); changed {
		s.list, s.list.at = s.list.narrowed(s.list.at, words)
	}
	return s
}

// leave gives c to the work. A session quits with its work alone, and a screen never does.
func (s pickScreen) leave(c Choice) (pickScreen, tea.Cmd) {
	s.left = true
	return s, func() tea.Msg {
		s.picks <- c
		return nil
	}
}

// enabled returns s with its boxes held while the scan ticks them, as a press would race it. A run
// waits for something chosen, as a run with no Only runs every step. The state screen waits for the
// scan's end, as both read the same machine.
func (s pickScreen) enabled() pickScreen {
	for place := pickTogglePlace; place <= pickPendingPlace; place++ {
		s.footer = s.footer.Enable(place, !s.scanning)
	}
	some := !s.scanning && len(s.list.only()) > 0
	s.footer = s.footer.Enable(pickRunPlace, some).Enable(pickCheckPlace, some)
	return s
}

// statusLine is the title's right side, of the steps alone, a queue's stages leaving out.
func (s pickScreen) statusLine() string {
	on, scanned, all := s.list.tally()
	return s.status(on, scanned, all, len(s.list.passedOver()) > 0)
}

// layout is the screen with its panes empty, as what fills them depends on their sizes. The list
// holds the focus alone, so that a window too short for two panes shows it.
func (s pickScreen) layout() Layout {
	return Layout{
		Title:  s.title,
		Status: s.statusLine(),
		Panes:  []Pane{{Title: "Steps", Focused: true}, {Title: "Detail"}},
		Footer: s.footer,
	}
}

// listHeight is the lines of the pane of steps that the list has.
func (s pickScreen) listHeight() int {
	height := s.layout().PaneSizes(s.window.Width, s.window.Height)[s.list.pane(s.list.side)].Height
	if s.filtering() {
		height--
	}
	return max(height, 0)
}

// View draws the screen.
func (s pickScreen) View() tea.View {
	l := s.layout()
	l.Panes[1].Body = s.detail(l.PaneSizes(s.window.Width, s.window.Height)[1], s.notes())
	l.Panes[0].Body = s.list.view(s.theme, s.listHeight(), s.list.at, true)
	if s.filtering() {
		l.Panes[0].Body = s.filter.line(s.theme) + "\n" + l.Panes[0].Body
	}
	return l.View(s.theme, s.window.Width, s.window.Height)
}
