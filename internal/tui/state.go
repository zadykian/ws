package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// The places of the state screen's buttons.
const (
	readPlace = iota
	leavePlace
)

var readButton = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "read again"))

// StateRead is the end of a read: what the machine reports, or why it can't be read.
type StateRead struct {
	Sections []StateSection
	Err      error
}

// StateScreen shows what a machine holds of what ws manages, a tab for each kind. It changes nothing.
type StateScreen struct {
	theme Theme
	title string
	// read asks for a report, which comes as a StateRead, and leave ends the screen. A screen is an
	// Update and a View alone, so that a command's run and the wizard's screen can each hold it.
	read, leave tea.Cmd
	window      Size
	footer      Footer
	// rows is each tab's, and tab the tab on show. It's an array so that a value of the screen shares
	// none of it with an older one.
	rows [tabCount][]stateRow
	tab  int
	// reading holds while a read runs, and reported once one has come. problem is why the last
	// failed, and warning the last that a read's steps gave.
	reading, reported bool
	problem, warning  string
}

// NewStateScreen returns the screen, which asks for a report as it starts.
func NewStateScreen(theme Theme, title string, read, leave tea.Cmd) StateScreen {
	footer := NewFooter(readButton, Keys.Quit).Enable(readPlace, false)
	return StateScreen{theme: theme, title: title, read: read, leave: leave, footer: footer, reading: true}
}

// Init asks for the first report.
func (s StateScreen) Init() tea.Cmd {
	return s.read
}

// Update takes a report, the window's size and the keys. Of a step's lines, a warning alone shows: a report
// says more than the others do.
func (s StateScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.window = Size{msg.Width, msg.Height}
	case StateRead:
		return s.took(msg), nil
	case Line:
		if msg.Kind == Warned {
			s.warning = firstLine(printable(msg.Text))
		}
	case tea.KeyPressMsg:
		return s.pressed(msg)
	}
	return s, nil
}

// took ends a read. One that failed keeps the rows on show: an old report beats none.
func (s StateScreen) took(msg StateRead) StateScreen {
	s.reading = false
	s.footer = s.footer.Enable(readPlace, true)
	s.problem = ""
	if msg.Err != nil {
		s.problem = firstLine(printable(msg.Err.Error()))
		return s
	}
	s.reported = true
	s.rows = tabRows(msg.Sections)
	return s
}

// pressed acts on a key: Tab first, as the footer would take it, then the buttons.
func (s StateScreen) pressed(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Next):
		return s.turned(1), nil
	case key.Matches(msg, Keys.Prev):
		return s.turned(-1), nil
	}
	var place int
	if s.footer, place = s.footer.Update(msg); place >= 0 {
		return s.pushed(place)
	}
	return s, nil
}

// pushed acts on a button.
func (s StateScreen) pushed(place int) (tea.Model, tea.Cmd) {
	if place == leavePlace {
		return s, s.leave
	}
	s.reading, s.warning = true, ""
	s.footer = s.footer.Enable(readPlace, false)
	return s, s.read
}

// turned moves along one ring of the tabs and the buttons, as Tab must both turn the tabs and reach
// a button.
func (s StateScreen) turned(step int) StateScreen {
	next := s.tab + step
	if s.footer.Focused() < 0 && next >= 0 && next < tabCount {
		s.tab = next
		return s
	}
	s.footer = s.footer.move(step)
	if s.footer.Focused() >= 0 {
		return s
	}
	s.tab = 0
	if step < 0 {
		s.tab = tabCount - 1
	}
	return s
}
