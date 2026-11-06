package tui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// noteRows is the most rows of what a save said that the screen keeps under the form.
const noteRows = 6

// configureJob is what the screen asks of its work. It travels on a channel and never as a message,
// as the edits can hold a secret.
type configureJob struct {
	place int
	sheet Sheet
	edits map[string]string
}

// savedMsg is the end of a save. Its sheet reads what the file holds.
type savedMsg struct {
	place int
	sheet Sheet
	err   error
}

// save checks the form, and hands its edits to the work. A refusal takes the focus, for the field to
// be mended.
func (s configureScreen) save() (tea.Model, tea.Cmd) {
	s.notes = nil
	form, ok := s.form().Check()
	s = s.with(form)
	if !ok {
		s.footer.focus = 0
		return s.fitted(), nil
	}
	s.saving = true
	job := configureJob{place: s.at, sheet: s.sheets[s.at], edits: form.Edits()}
	return s.fitted(), func() tea.Msg { s.send(job); return nil }
}

// saved ends a save. A failed one keeps the edits, and shows why.
func (s configureScreen) saved(msg savedMsg) configureScreen {
	s.saving = false
	if msg.err != nil {
		s.notes = slices.Concat(s.notes, []Line{{Warned, msg.err.Error()}})
		return s.fitted()
	}
	s.sheets, s.forms = slices.Clone(s.sheets), slices.Clone(s.forms)
	s.sheets[msg.place], s.forms[msg.place] = msg.sheet, s.forms[msg.place].Refill(msg.sheet.Fields)
	return s.fitted()
}

// noteLines draws the end of what the last save said, at width.
func (s configureScreen) noteLines(width int) []string {
	var lines []string
	for _, note := range s.notes {
		style := s.theme.Muted
		if note.Kind == Warned {
			style = s.theme.Bad
		}
		lines = append(lines, strings.Split(style.Width(max(width, 1)).Render(cleanLines(note.Text)), "\n")...)
	}
	return lines[max(len(lines)-noteRows, 0):]
}

// split shares a pane between the form and the notes: it returns the form's room, and what the body
// holds under the form. The form keeps a row, for the line that takes the keys, and the notes keep
// their end, which says why. The blank over them goes before a row of theirs.
func (s configureScreen) split(pane Size) (Size, string) {
	notes := s.noteLines(pane.Width)
	notes = notes[max(len(notes)-max(pane.Height-1, 0), 0):]
	if len(notes) == 0 {
		return pane, ""
	}
	gap := "\n\n"
	pane.Height -= len(notes) + 1
	if pane.Height < 1 {
		pane.Height, gap = 1, "\n"
	}
	return pane, gap + strings.Join(notes, "\n")
}
