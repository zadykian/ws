package tui

import (
	"context"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// noteRows is the most rows of what a save said that the screen keeps under the form.
const noteRows = 6

// Configure draws the sheets on c.Out, a form each, until a leave, with title over them. It returns
// nil past a leave, and Cut's error past Ctrl-C or a signal, which drop what is unsaved. Saves run
// here, off the screen's goroutine, and what they print stays on the screen.
func Configure(ctx context.Context, c Config, title string, sheets ...Sheet) error {
	ctx, end := context.WithCancelCause(ctx)
	defer end(nil)
	a := newAsks(end)
	send := func(j configureJob) {
		a.ask(func(ctx context.Context, b *Bridge) tea.Msg {
			sheet, err := j.run(ctx, b)
			return savedMsg{j.place, sheet, err}
		})
	}
	screen := newConfigureScreen(title, c.Theme, send, a.leave, sheets)
	return Run(ctx, c, screen, a.serve)
}

// configureJob is what the screen asks of its work. It travels inside an ask's closure and never as a
// message, as the edits can hold a secret.
type configureJob struct {
	place int
	sheet Sheet
	edits map[string]string
}

// run reads the sheet where it waits for that, and saves the edits otherwise.
func (j configureJob) run(ctx context.Context, b *Bridge) (Sheet, error) {
	if j.sheet.Open != nil {
		return j.sheet.Open(ctx, b)
	}
	return j.sheet.Save(ctx, b, j.edits)
}

// savedMsg is the end of a read or of a save. Its sheet reads what the file holds.
type savedMsg struct {
	place int
	sheet Sheet
	err   error
}

// leave ends the screen, and asks first where it would lose an edit.
func (s configureScreen) leave(again bool) (tea.Model, tea.Cmd) {
	if !again && s.unsaved() > 0 {
		s.leaving = true
		return s, nil
	}
	return s, s.end
}

// save checks the form, and hands its edits to the work. A refusal takes the focus, for the field to
// be mended.
func (s configureScreen) save() (tea.Model, tea.Cmd) {
	if s.sheets[s.at].Open != nil {
		return s.opened()
	}
	s.notes = nil
	form, ok := s.form().Check()
	s = s.with(form)
	if !ok {
		s.footer.Focus = 0
		return s.fitted(), nil
	}
	s.saving = true
	job := configureJob{place: s.at, sheet: s.sheets[s.at], edits: form.Edits()}
	return s.fitted(), func() tea.Msg { s.send(job); return nil }
}

// opened hands the work the read of the sheet that shows, where it waits for one. The keys wait for
// the read as for a save, and a question goes to the session, as in a save.
func (s configureScreen) opened() (tea.Model, tea.Cmd) {
	if s.sheets[s.at].Open == nil {
		return s.fitted(), nil
	}
	s.notes, s.saving = nil, true
	job := configureJob{place: s.at, sheet: s.sheets[s.at]}
	return s.fitted(), func() tea.Msg { s.send(job); return nil }
}

// saved ends a read or a save. A failed one keeps the edits, and shows why.
func (s configureScreen) saved(msg savedMsg) configureScreen {
	s.saving = false
	if msg.err != nil {
		s.notes = slices.Concat(s.notes, []widget.Line{{Kind: widget.Warned, Text: msg.err.Error()}})
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
		if note.Kind == widget.Warned {
			style = s.theme.Bad
		}
		lines = append(lines, strings.Split(style.Width(max(width, 1)).Render(cleanLines(note.Text)), "\n")...)
	}
	return lines[max(len(lines)-noteRows, 0):]
}

// split shares a pane between the form and the notes: it returns the form's room, and what the body
// holds under the form. The form keeps a row, for the line that takes the keys, and the notes keep
// their end, which says why. The blank over them goes before a row of theirs.
func (s configureScreen) split(pane widget.Size) (widget.Size, string) {
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
