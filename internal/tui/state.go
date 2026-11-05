package tui

import (
	tea "charm.land/bubbletea/v2"
)

// StateRead is the end of a read: what the machine reports, or why it can't be read.
type StateRead struct {
	Sections []StateSection
	Err      error
}

// StateScreen shows what a machine holds of what ws manages, a tab for each kind. It changes nothing.
type StateScreen struct {
	theme Theme
	title string
	// read asks for a report, which comes as a StateRead. A screen is an Update and a View alone, so
	// that a command's run and the wizard's screen can each hold it.
	read   tea.Cmd
	window Size
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
func NewStateScreen(theme Theme, title string, read tea.Cmd) StateScreen {
	return StateScreen{theme: theme, title: title, read: read, reading: true}
}

// Init asks for the first report.
func (s StateScreen) Init() tea.Cmd {
	return s.read
}

// Update takes a report and the window's size. Of a step's lines, a warning alone shows: a report
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
	}
	return s, nil
}

// took ends a read. One that failed keeps the rows on show: an old report beats none.
func (s StateScreen) took(msg StateRead) StateScreen {
	s.reading = false
	s.problem = ""
	if msg.Err != nil {
		s.problem = firstLine(printable(msg.Err.Error()))
		return s
	}
	s.reported = true
	s.rows = tabRows(msg.Sections)
	return s
}
