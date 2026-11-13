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
	filterPlace = iota
	readPlace
	leavePlace
)

var (
	filterButton = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
	readButton   = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "read again"))
	// openKey and backKey move between the list and the detail pane, as Tab moves between the tabs.
	openKey = key.NewBinding(key.WithKeys("enter", "right", "l"))
	backKey = key.NewBinding(key.WithKeys("esc", "left", "h"))
)

// StateRead is the end of a read: what the machine reports, or why it can't be read.
type StateRead struct {
	Sections []StateSection
	Err      error
}

// StateScreen shows what a machine holds of what ws manages. It has a tab for each kind, the rows
// of the tab on show, and a pane of the selected row's wanted and found state. It changes nothing.
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
	// inDetail gives the keys to the detail pane, and scroll is the first of its lines on show.
	inDetail bool
	scroll   int
	// reading holds while a read runs, and reported once one has come. problem is why the last
	// failed, and warning the last that a read's steps gave.
	reading, reported bool
	problem, warning  string
	// filter holds the words that a row's name must contain.
	filter filter
}

// NewStateScreen returns the screen, which asks for a report as it starts.
func NewStateScreen(theme Theme, title string, read, leave tea.Cmd) StateScreen {
	footer := NewFooter(filterButton, readButton, Keys.Quit).Enable(readPlace, false)
	return StateScreen{
		theme: theme, title: title, read: read, leave: leave, footer: footer, reading: true, filter: newFilter(),
	}
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
	case tea.PasteMsg:
		if s.filter.editing {
			return s.edited(msg), nil
		}
	case tea.KeyPressMsg:
		if s.filter.editing {
			return s.edited(msg), nil
		}
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
	picks := s.picks()
	s.reported = true
	s.rows = tabRows(msg.Sections)
	return s.refound(picks)
}

// picks returns the row that each tab has selected, nil where a tab shows none. Whatever moves the
// rows takes them first. The input rewrites its words in place, so a copy of the screen from before
// an edit reads the new words. Its cursors don't fit the rows those leave.
func (s StateScreen) picks() [tabCount]*stateRow {
	var picks [tabCount]*stateRow
	for tab, cursor := range s.cursors {
		if rows := s.shown(tab); len(rows) > 0 {
			picks[tab] = &rows[cursor]
		}
	}
	return picks
}

// refound puts each tab's selection back on its row, once a read or the filter moved the rows under
// it. Where the row went, the selection keeps its place.
func (s StateScreen) refound(picks [tabCount]*stateRow) StateScreen {
	for tab, picked := range picks {
		rows := s.shown(tab)
		s.cursors[tab] = max(min(s.cursors[tab], len(rows)-1), 0)
		if picked == nil {
			continue
		}
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
		s.inDetail = len(s.shown(s.tab)) > 0
		return s, nil
	case s.inDetail && key.Matches(msg, backKey):
		s.inDetail = false
		return s, nil
	case key.Matches(msg, clearKey):
		return s.cleared(), nil
	}
	return s.moved(msg), nil
}

// pushed acts on a button.
func (s StateScreen) pushed(place int) (tea.Model, tea.Cmd) {
	switch place {
	case filterPlace:
		s.filter, s.inDetail = s.filter.opened(), false
		return s, nil
	case leavePlace:
		return s, s.leave
	}
	s.reading, s.warning = true, ""
	s.footer = s.footer.Enable(readPlace, false)
	return s, s.read
}

// turned moves along one ring of the tabs and the buttons, as Tab must both turn the tabs and reach
// a button. The list takes the keys in a new tab.
func (s StateScreen) turned(step int) StateScreen {
	s.inDetail, s.scroll = false, 0
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

// moved moves the selection, or the detail pane's lines where that pane has the keys. A new row
// shows from its first line.
func (s StateScreen) moved(msg tea.KeyPressMsg) StateScreen {
	sizes := s.layout().PaneSizes(s.window.Width, s.window.Height)
	if s.inDetail {
		size := sizes[detailPane]
		last := max(len(s.detailLines(size.Width))-size.Height, 0)
		s.scroll = stepped(msg, s.scroll, size.Height, last)
		return s
	}
	cursor := stepped(msg, s.cursors[s.tab], sizes[listPane].Height, len(s.shown(s.tab))-1)
	if cursor != s.cursors[s.tab] {
		s.cursors[s.tab], s.scroll = cursor, 0
	}
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
