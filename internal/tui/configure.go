package tui

import (
	"context"
	"fmt"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// The places of the Configure screen's buttons.
const (
	savePlace = iota
	formLeavePlace
)

var (
	saveButton  = key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save"))
	leaveButton = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "leave"))
	// fieldUp and fieldDown are the arrows alone, as k and j are letters of a value.
	fieldUp   = key.NewBinding(key.WithKeys("up"))
	fieldDown = key.NewBinding(key.WithKeys("down"))
)

// Sheet is a file that the Configure screen edits.
type Sheet struct {
	// Name titles the file's form.
	Name string
	// Fields gives the fields that show under the edits so far. It is pure, as it runs on each edit.
	Fields func(edits map[string]string) []FieldSpec
	// Save writes the edits on the work's goroutine, and returns the sheet of what the file then holds.
	Save func(ctx context.Context, b *Bridge, edits map[string]string) (Sheet, error)
}

// configureScreen edits its sheets, a form each, with one form on the screen.
type configureScreen struct {
	title  string
	theme  Theme
	sheets []Sheet
	forms  []Form
	// at is the place of the sheet that shows.
	at     int
	footer Footer
	window Size
	// notes is what the last save said. send hands the work a job, and saving holds the keys till it
	// ends, as the form would change under the file that it writes.
	notes  []Line
	send   func(configureJob)
	saving bool
}

// newConfigureScreen returns the screen of sheets, on the first.
func newConfigureScreen(title string, theme Theme, send func(configureJob), sheets []Sheet) configureScreen {
	forms := make([]Form, 0, len(sheets))
	for _, sheet := range sheets {
		forms = append(forms, NewForm(theme, sheet.Fields))
	}
	footer := NewFooter(saveButton, leaveButton)
	return configureScreen{title: title, theme: theme, sheets: sheets, forms: forms, footer: footer, send: send}
}

// Init starts nothing, as the screen waits for a key.
func (s configureScreen) Init() tea.Cmd { return nil }

// Update takes the window's size, the keys, a paste, and what a save said and how it ended.
func (s configureScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.window = Size{msg.Width, msg.Height}
		return s.fitted(), nil
	case Line:
		s.notes = slices.Concat(s.notes, []Line{msg})
		return s.fitted(), nil
	case savedMsg:
		return s.saved(msg), nil
	case tea.KeyPressMsg:
		return s.pressed(msg)
	case tea.PasteMsg:
		if s.footer.Focused() < 0 && !s.saving {
			return s.with(s.form().Update(msg)), nil
		}
	}
	return s, nil
}

// form is the form that shows.
func (s configureScreen) form() Form { return s.forms[s.at] }

// with replaces the form that shows.
func (s configureScreen) with(form Form) configureScreen {
	s.forms = slices.Clone(s.forms)
	s.forms[s.at] = form
	return s
}

// fitted sizes the forms to the pane, less what the notes take.
func (s configureScreen) fitted() configureScreen {
	size, _ := s.split(s.layout().PaneSizes(s.window.Width, s.window.Height)[0])
	s.forms = slices.Clone(s.forms)
	for place, form := range s.forms {
		s.forms[place] = form.Fit(size)
	}
	return s
}

// fieldStep is the way that msg moves along a form's fields, or 0.
func fieldStep(msg tea.KeyPressMsg) int {
	switch {
	case key.Matches(msg, Keys.Next, Keys.Press, fieldDown):
		return 1
	case key.Matches(msg, Keys.Prev, fieldUp):
		return -1
	}
	return 0
}

// pressed moves the focus along the one ring of the fields and the buttons, presses a button, or
// hands a field the key. The keys wait for a save.
func (s configureScreen) pressed(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if s.saving {
		return s, nil
	}
	onForm := s.footer.Focused() < 0
	if step := fieldStep(msg); onForm && step != 0 {
		return s.stepped(step), nil
	}
	var place int
	s.footer, place = s.footer.Update(msg)
	switch {
	case place == savePlace:
		return s.save()
	case !onForm && s.footer.Focused() < 0:
		return s.with(s.form().Enter(key.Matches(msg, Keys.Prev))), nil
	case onForm:
		return s.with(s.form().Update(msg)), nil
	}
	return s, nil
}

// stepped moves the focus along the fields, and past either end onto the buttons.
func (s configureScreen) stepped(step int) configureScreen {
	form, moved := s.form().Move(step)
	if moved {
		return s.with(form)
	}
	s.footer = s.footer.move(step)
	return s.with(form.Blur())
}

// layout is the screen with its pane empty, as what fills it depends on its size.
func (s configureScreen) layout() Layout {
	at, of := s.form().Place()
	return Layout{
		Title:  s.title,
		Panes:  []Pane{{Title: fmt.Sprintf("%s %d/%d", s.sheets[s.at].Name, at, of), Focused: s.footer.Focused() < 0}},
		Footer: s.footer,
	}
}

// View draws the form that shows.
func (s configureScreen) View() tea.View {
	l := s.layout()
	_, under := s.split(l.PaneSizes(s.window.Width, s.window.Height)[0])
	l.Panes[0].Body = s.form().Render() + under
	return l.View(s.theme, s.window.Width, s.window.Height)
}
