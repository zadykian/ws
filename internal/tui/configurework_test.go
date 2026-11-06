package tui

import (
	"errors"
	"maps"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var ctrlS = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}

// recording is a screen whose jobs land in jobs.
func recording(jobs *[]configureJob, sheets ...Sheet) configureScreen {
	s := testConfigure(sheets...)
	s.send = func(j configureJob) { *jobs = append(*jobs, j) }
	return s
}

func TestConfigureSaveHandsTheWorkTheEdits(t *testing.T) {
	var jobs []configureJob
	s := fed(recording(&jobs, wayFile()), enter, tea.PasteMsg{Content: "2222"}, ctrlS)
	if len(jobs) != 1 || !maps.Equal(jobs[0].edits, map[string]string{"PORT": "2222"}) {
		t.Fatalf("jobs = %+v", jobs)
	}
	if s = fed(s, press('x'), tea.PasteMsg{Content: "y"}); s.form().Edits()["PORT"] != "2222" {
		t.Errorf("a key reached the form in a save: %v", s.form().Edits())
	}
}

func TestConfigureRefusedSaveStaysOnTheField(t *testing.T) {
	refused := errors.New("no such host")
	sheet := Sheet{Name: "way.conf", Fields: func(map[string]string) []FieldSpec {
		return []FieldSpec{{Label: "A"}, {Label: "HOST", Valid: func(string) error { return refused }}}
	}}
	var jobs []configureJob
	s := fed(recording(&jobs, sheet), tab, tab, ctrlS)
	if len(jobs) != 0 || s.saving {
		t.Errorf("jobs = %+v, saving = %v", jobs, s.saving)
	}
	if s.footer.Focused() >= 0 || s.form().name() != "HOST" {
		t.Errorf("button %d, field %q", s.footer.Focused(), s.form().name())
	}
}
