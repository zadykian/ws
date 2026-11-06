package tui

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
)

// Sheet is a file that the Configure screen edits.
type Sheet struct {
	// Name titles the file's form.
	Name string
	// Fields gives the fields that show under the edits so far. It is pure, as it runs on each edit.
	Fields func(edits map[string]string) []FieldSpec
}

// configureScreen edits its sheets, a form each, with one form on the screen.
type configureScreen struct {
	title  string
	theme  Theme
	sheets []Sheet
	forms  []Form
	// at is the place of the sheet that shows.
	at     int
	window Size
}

// newConfigureScreen returns the screen of sheets, on the first.
func newConfigureScreen(title string, theme Theme, sheets []Sheet) configureScreen {
	forms := make([]Form, 0, len(sheets))
	for _, sheet := range sheets {
		forms = append(forms, NewForm(theme, sheet.Fields))
	}
	return configureScreen{title: title, theme: theme, sheets: sheets, forms: forms}
}

// Init starts nothing, as the screen waits for a key.
func (s configureScreen) Init() tea.Cmd { return nil }

// Update takes the window's size.
func (s configureScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		s.window = Size{msg.Width, msg.Height}
		return s.fitted(), nil
	}
	return s, nil
}

// form is the form that shows.
func (s configureScreen) form() Form { return s.forms[s.at] }

// fitted sizes the forms to the pane.
func (s configureScreen) fitted() configureScreen {
	size := s.layout().PaneSizes(s.window.Width, s.window.Height)[0]
	s.forms = slices.Clone(s.forms)
	for place, form := range s.forms {
		s.forms[place] = form.Fit(size)
	}
	return s
}

// layout is the screen with its pane empty, as what fills it depends on its size.
func (s configureScreen) layout() Layout {
	at, of := s.form().Place()
	return Layout{
		Title: s.title,
		Panes: []Pane{{Title: fmt.Sprintf("%s %d/%d", s.sheets[s.at].Name, at, of), Focused: true}},
	}
}

// View draws the form that shows.
func (s configureScreen) View() tea.View {
	l := s.layout()
	l.Panes[0].Body = s.form().Render()
	return l.View(s.theme, s.window.Width, s.window.Height)
}
