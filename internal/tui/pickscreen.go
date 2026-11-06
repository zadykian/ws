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
	pickConfigurePlace
)

// pickClearKey drops the filter's words.
var pickClearKey = key.NewBinding(key.WithKeys("esc"))

// Choice is what the list leaves to a run. No steps says that nothing runs.
type Choice struct {
	// Only names the steps as Options.Only does.
	Only []string
	// Check plans them and applies nothing.
	Check bool
	// Configure leaves a queue for its values' form, whatever else is chosen.
	Configure bool
	// Partial gives, by stage of a queue, the steps on where some of the stage's are off, by entry.
	// A stage it has no key for runs whole.
	Partial map[string][]string
	// Whole names the stages of a queue that run every step, one of them by a tick. Such a stage is
	// to run though its last run passed. One that a lock alone holds whole isn't.
	Whole []string
	// list is the lists as the maintainer left them, for the screen that follows the run.
	list pickList
}

// scannedMsg ends the scan. problem says why it stopped short, if it did.
type scannedMsg struct{ problem string }

// reportedMsg says what a stage's machine reports, or why it reports nothing.
type reportedMsg struct {
	stage    string
	sections []StateSection
	problem  string
}

// pickScreen is the list that a run is chosen from. A scan fills it behind the screen, and the
// maintainer can read and move as it goes.
type pickScreen struct {
	title  string
	theme  Theme
	list   pickList
	footer Footer
	window Size
	// scanning says that events of the scan are still on their way.
	scanning bool
	// problem is why the scan stopped short.
	problem string
	// editing says that the keys go to the filter's line.
	editing bool
	// picks takes the choice, and holds one, so that sending it waits for nobody.
	picks chan<- Choice
	// left says that the choice has gone, and the screen waits for the work to end it.
	left bool
	// queued draws the list as a queue's two sides.
	queued bool
	// unread is why a machine reported nothing, and its steps stand as the queue named them.
	unread string
}

// newPickScreen returns the screen of groups, its scan under way.
func newPickScreen(title string, theme Theme, groups []Group, picks chan<- Choice) pickScreen {
	button := func(k, label string) key.Binding { return key.NewBinding(key.WithKeys(k), key.WithHelp(k, label)) }
	footer := NewFooter(button("space", "toggle"), button("a", "all"), button("n", "none"), button("i", "invert"),
		button("p", "pending"), button("/", "filter"), button("enter", "run"), button("c", "check"), Keys.Quit)
	s := pickScreen{title: title, theme: theme, list: newPickList(groups), footer: footer, scanning: true, picks: picks}
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
		s.scanning, s.problem = false, msg.problem
		if s.queued {
			s.list = s.list.opened()
		}
	case reportedMsg:
		s.unread = msg.problem
		if msg.problem == "" {
			s.list = s.list.reported(msg.stage, msg.sections)
		}
	case tea.WindowSizeMsg:
		s.window = Size{msg.Width, msg.Height}
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
	case s.editing:
		return s.typed(msg), nil
	case key.Matches(msg, pickClearKey):
		s.list = s.list.filtered("")
		return s, nil
	case s.queued && key.Matches(msg, Keys.Next):
		return s.tabbed(1), nil
	case s.queued && key.Matches(msg, Keys.Prev):
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
		s.editing = true
	case pickRunPlace:
		return s.leave(s.choice(false))
	case pickCheckPlace:
		return s.leave(s.choice(true))
	case pickQuitPlace:
		return s.leave(Choice{})
	case pickConfigurePlace:
		return s.leave(Choice{Configure: true})
	default:
		s.list = s.list.moved(msg, max(s.listHeight(), 1))
	}
	return s, nil
}

// typed edits the filter, which narrows the list as it goes. Enter keeps it, and Esc drops it.
func (s pickScreen) typed(msg tea.KeyPressMsg) pickScreen {
	words := []rune(s.list.words)
	switch {
	case key.Matches(msg, Keys.Press):
		s.editing = false
	case key.Matches(msg, pickClearKey):
		s.editing = false
		s.list = s.list.filtered("")
	case msg.Code == tea.KeyBackspace:
		s.list = s.list.filtered(string(words[:max(len(words)-1, 0)]))
	case msg.Text != "":
		s.list = s.list.filtered(s.list.words + msg.Text)
	}
	return s
}

// choice is what the list holds, to run or to check.
func (s pickScreen) choice(check bool) Choice {
	return Choice{Only: s.list.only(), Check: check, Partial: s.list.partial(), Whole: s.list.ticked(), list: s.list}
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
// waits for something chosen, as a run with no Only runs every step.
func (s pickScreen) enabled() pickScreen {
	for place := pickTogglePlace; place <= pickPendingPlace; place++ {
		s.footer = s.footer.Enable(place, !s.scanning)
	}
	some := !s.scanning && len(s.list.only()) > 0
	s.footer = s.footer.Enable(pickRunPlace, some).Enable(pickCheckPlace, some)
	return s
}

// status is the title's right side: how far the scan is, then how much is chosen, of a queue's stages
// alone. Its mark stands for the warning of a role passed over, which a short window has no pane for.
func (s pickScreen) status() string {
	on, all := s.list.count(func(r pickRow) bool { return r.stage == "" })
	if s.left {
		return "leaving"
	}
	if s.scanning {
		_, scanned := s.list.count(func(r pickRow) bool { return r.stage == "" && r.state != stepOpen })
		return fmt.Sprintf("scanning %d/%d", scanned, all)
	}
	if len(s.list.passedOver()) > 0 {
		return fmt.Sprintf("! %d of %d chosen", on, all)
	}
	return fmt.Sprintf("%d of %d chosen", on, all)
}

// layout is the screen with its panes empty, as what fills them depends on their sizes. The list
// holds the focus alone, so that a window too short for two panes shows it.
func (s pickScreen) layout() Layout {
	if s.queued {
		return s.queueLayout()
	}
	return Layout{
		Title:  s.title,
		Status: s.status(),
		Panes:  []Pane{{Title: "Steps", Focused: true}, {Title: "Detail"}},
		Footer: s.footer,
	}
}

// filtering says that the filter's line shows, over the list.
func (s pickScreen) filtering() bool {
	return s.editing || s.list.words != ""
}

// filterLine draws the filter. The block marks it as the place that takes the keys.
func (s pickScreen) filterLine() string {
	if s.editing {
		return "/ " + s.list.words + s.theme.ButtonFocus.Render(" ")
	}
	return s.theme.Muted.Render("/ " + s.list.words)
}

// listHeight is the lines of the pane of steps that the list has.
func (s pickScreen) listHeight() int {
	height := s.layout().PaneSizes(s.window.Width, s.window.Height)[s.list.side].Height
	if s.filtering() {
		height--
	}
	return max(height, 0)
}

// View draws the screen.
func (s pickScreen) View() tea.View {
	if s.queued {
		return s.queueView()
	}
	l := s.layout()
	l.Panes[1].Body = s.detail(l.PaneSizes(s.window.Width, s.window.Height)[1])
	l.Panes[0].Body = s.list.view(s.theme, s.listHeight())
	if s.filtering() {
		l.Panes[0].Body = s.filterLine() + "\n" + l.Panes[0].Body
	}
	return l.View(s.theme, s.window.Width, s.window.Height)
}
