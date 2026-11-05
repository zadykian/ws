package tui

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// The panes of the state screen.
const (
	listPane = iota
	detailPane
)

// The places of the state screen's buttons.
const (
	readPlace = iota
	leavePlace
)

var (
	readButton = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "read again"))
	// openKey and backKey move between the list and the detail pane, as Tab moves between the tabs.
	openKey = key.NewBinding(key.WithKeys("enter", "right", "l"))
	backKey = key.NewBinding(key.WithKeys("esc", "left", "h"))
)

// StateRead is the end of a read: what the machine reports, or why it can't be read.
type StateRead struct {
	Sections []StateSection
	Err      error
}

// StateScreen shows what a machine holds of what ws manages: a tab for each kind, the rows of the
// tab on show, and a pane of the selected row's wanted and found state. It changes nothing.
type StateScreen struct {
	theme Theme
	title string
	// read asks for a report, which comes as a StateRead, and leave ends the screen. A screen is an
	// Update and a View alone, so that a command's run and the wizard's screen can each hold it.
	read, leave tea.Cmd
	window      Size
	footer      Footer
	// rows is each tab's, cursors the selected row of each, and tab the tab on show. They are arrays
	// so that a value of the screen shares none of them with an older one.
	rows    [tabCount][]stateRow
	cursors [tabCount]int
	tab     int
	// inDetail gives the keys to the detail pane.
	inDetail bool
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
	was := s
	s.reported = true
	s.rows = tabRows(msg.Sections)
	return s.refound(was)
}

// refound puts each tab's selection back on its row, once a read moved the rows under it. Where the
// row went, the selection keeps its place.
func (s StateScreen) refound(was StateScreen) StateScreen {
	for tab := range s.cursors {
		rows := s.rows[tab]
		s.cursors[tab] = max(min(was.cursors[tab], len(rows)-1), 0)
		old := was.rows[tab]
		if len(old) == 0 {
			continue
		}
		picked := old[was.cursors[tab]]
		same := func(row stateRow) bool { return row.step == picked.step && row.name == picked.name }
		if at := slices.IndexFunc(rows, same); at >= 0 {
			s.cursors[tab] = at
		}
	}
	return s
}

// pressed acts on a key: Tab first, as the footer would take it, then the buttons, then the panes.
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
	switch {
	case key.Matches(msg, openKey):
		s.inDetail = len(s.rows[s.tab]) > 0
		return s, nil
	case key.Matches(msg, backKey):
		s.inDetail = false
		return s, nil
	}
	return s.moved(msg), nil
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
// a button. The list takes the keys in a new tab.
func (s StateScreen) turned(step int) StateScreen {
	s.inDetail = false
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

// moved moves the selection, while the list has the keys.
func (s StateScreen) moved(msg tea.KeyPressMsg) StateScreen {
	if s.inDetail {
		return s
	}
	sizes := s.layout().PaneSizes(s.window.Width, s.window.Height)
	s.cursors[s.tab] = stepped(msg, s.cursors[s.tab], sizes[listPane].Height, len(s.rows[s.tab])-1)
	return s
}

// stepped returns where a key moves a place in 0 to last, by a page of page lines.
func stepped(msg tea.KeyPressMsg, at, page, last int) int {
	switch {
	case key.Matches(msg, Keys.Up):
		at--
	case key.Matches(msg, Keys.Down):
		at++
	case key.Matches(msg, Keys.PageUp):
		at -= page
	case key.Matches(msg, Keys.PageDown):
		at += page
	case key.Matches(msg, Keys.Top):
		at = 0
	case key.Matches(msg, Keys.Bottom):
		at = last
	}
	return max(min(at, last), 0)
}
