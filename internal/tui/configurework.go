package tui

import tea "charm.land/bubbletea/v2"

// configureJob is what the screen asks of its work. It travels on a channel and never as a message,
// as the edits can hold a secret.
type configureJob struct {
	place int
	sheet Sheet
	edits map[string]string
}

// save checks the form, and hands its edits to the work. A refusal takes the focus, for the field to
// be mended.
func (s configureScreen) save() (tea.Model, tea.Cmd) {
	form, ok := s.form().Check()
	s = s.with(form)
	if !ok {
		s.footer.focus = 0
		return s, nil
	}
	s.saving = true
	job := configureJob{place: s.at, sheet: s.sheets[s.at], edits: form.Edits()}
	return s, func() tea.Msg { s.send(job); return nil }
}
