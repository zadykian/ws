package tui

import (
	"errors"
	"maps"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var (
	ctrlS = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	ctrlN = tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl}
)

// recording is a screen whose jobs land in jobs.
func recording(jobs *[]configureJob, ends *int, sheets ...Sheet) configureScreen {
	s := testConfigure(sheets...)
	s.send = func(j configureJob) { *jobs = append(*jobs, j) }
	s.end = func() tea.Msg { *ends++; return nil }
	return s
}

func TestConfigureSaveHandsTheWorkTheEdits(t *testing.T) {
	var jobs []configureJob
	s := fed(recording(&jobs, nil, wayFile()), enter, tea.PasteMsg{Content: "2222"}, ctrlS)
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
	s := fed(recording(&jobs, nil, sheet), tab, tab, ctrlS)
	if len(jobs) != 0 || s.saving {
		t.Errorf("jobs = %+v, saving = %v", jobs, s.saving)
	}
	if s.footer.Focused() >= 0 || s.form().name() != "HOST" {
		t.Errorf("button %d, field %q", s.footer.Focused(), s.form().name())
	}
}

func TestConfigureSavedStartsTheFormOverOnTheFile(t *testing.T) {
	var jobs []configureJob
	s := fed(recording(&jobs, nil, wayFile()), enter, tea.PasteMsg{Content: "2222"}, ctrlS)
	wrote := Sheet{Name: "way.conf", Fields: func(map[string]string) []FieldSpec {
		return []FieldSpec{{Label: "HOST"}, {Label: "PORT", Value: "2222"}}
	}}
	s = fed(s, Line{Said, "wrote way.conf"}, savedMsg{sheet: wrote})
	if s.saving || len(s.form().Edits()) != 0 || s.form().name() != "PORT" {
		t.Errorf("saving = %v, Edits = %v, field %q", s.saving, s.form().Edits(), s.form().name())
	}
	if got := stripped(s.View().Content); !strings.Contains(got, "wrote way.conf") {
		t.Errorf("no note:\n%s", got)
	}
	if s = fed(s, ctrlS); len(s.notes) != 0 {
		t.Errorf("a save kept the last one's notes: %v", s.notes)
	}
}

func TestConfigureFailedSaveKeepsTheEditsAndSaysWhy(t *testing.T) {
	var jobs []configureJob
	s := fed(recording(&jobs, nil, wayFile()), enter, tea.PasteMsg{Content: "2222"}, ctrlS)
	s = fed(s, savedMsg{err: errors.New("the file is read-only")})
	if s.saving || s.form().Edits()["PORT"] != "2222" {
		t.Errorf("saving = %v, Edits = %v", s.saving, s.form().Edits())
	}
	if got := stripped(s.View().Content); !strings.Contains(got, "the file is read-only") {
		t.Errorf("no reason:\n%s", got)
	}
}

func TestConfigureKeepsTheEndOfWhatASaveSaid(t *testing.T) {
	s := testConfigure(wayFile())
	for range noteRows + 2 {
		s = fed(s, Line{Said, "a line"})
	}
	if got := len(s.noteLines(40)); got != noteRows {
		t.Errorf("%d rows", got)
	}
}

func TestConfigureShortWindowKeepsTheInputAndTheNotesEnd(t *testing.T) {
	refusal := strings.Repeat("no room here ", 30) + "why"
	for _, height := range []int{5, 6, 8, 9, 11} {
		s := fed(testConfigure(wayFile()), tea.WindowSizeMsg{Width: 60, Height: height}, enter)
		s = fed(s, tea.PasteMsg{Content: "2222"}, Line{Warned, refusal})
		got := stripped(s.View().Content)
		// A pane of one row holds the input alone. The status still counts the edit.
		if !strings.Contains(got, "2222") || height > 5 && !strings.Contains(got, "why") {
			t.Errorf("height %d drew:\n%s", height, got)
		}
	}
}

func TestConfigureLeavesAtOnceWithNoEdit(t *testing.T) {
	var jobs []configureJob
	var ends int
	fed(recording(&jobs, &ends, wayFile()), esc)
	if len(jobs) != 0 || ends != 1 {
		t.Errorf("jobs = %+v, ends = %d", jobs, ends)
	}
}

func TestConfigureAsksBeforeItLosesAnEdit(t *testing.T) {
	var jobs []configureJob
	var ends int
	s := fed(recording(&jobs, &ends, wayFile()), enter, tea.PasteMsg{Content: "2222"})
	if got := stripped(s.View().Content); !strings.Contains(got, "1 unsaved") {
		t.Errorf("no count:\n%s", got)
	}
	s = fed(s, tea.WindowSizeMsg{Width: 30, Height: 24}, esc)
	if got := stripped(s.View().Content); len(jobs) != 0 || !strings.Contains(got, "esc again") {
		t.Errorf("jobs = %+v, drew:\n%s", jobs, got)
	}
	if s = fed(s, press('1')); s.leaving {
		t.Error("a key kept the question")
	}
	fed(s, esc, esc)
	if len(jobs) != 0 || ends != 1 {
		t.Errorf("jobs = %+v, ends = %d", jobs, ends)
	}
}

func TestConfigureWalksItsSheetsAndKeepsTheirTyping(t *testing.T) {
	s := testConfigure(wayFile(), Sheet{Name: "other.conf", Fields: longSpecs})
	if got := stripped(s.View().Content); !strings.Contains(got, "[ctrl+n] next form") {
		t.Errorf("no button:\n%s", got)
	}
	s = fed(s, enter, tea.PasteMsg{Content: "2222"}, ctrlN)
	if got := stripped(s.View().Content); !strings.Contains(got, "other.conf 1/6") {
		t.Errorf("no second form:\n%s", got)
	}
	s = fed(s, shiftTab, enter)
	if s.at != 0 || s.form().Edits()["PORT"] != "2222" || s.footer.Focused() >= 0 {
		t.Errorf("at = %d, Edits = %v, button %d", s.at, s.form().Edits(), s.footer.Focused())
	}
}
