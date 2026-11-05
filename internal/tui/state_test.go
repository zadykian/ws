package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// readAsked is the message of a test's read.
type readAsked struct{}

var sized = tea.WindowSizeMsg{Width: 80, Height: 24}

func newTestState() StateScreen {
	return NewStateScreen(testTheme, "ws server state", func() tea.Msg { return readAsked{} })
}

// feedState gives s each message, and returns the last Cmd.
func feedState(t *testing.T, s StateScreen, msgs ...tea.Msg) (StateScreen, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		var model tea.Model
		model, cmd = s.Update(msg)
		next, ok := model.(StateScreen)
		if !ok {
			t.Fatalf("Update returned a %T", model)
		}
		s = next
	}
	return s, cmd
}

// reportedState is the screen in an 80 by 24 window, after the report of testSections.
func reportedState(t *testing.T) StateScreen {
	s, _ := feedState(t, newTestState(), sized, StateRead{Sections: testSections()})
	return s
}

func TestStateScreenAsksForAReportAsItStarts(t *testing.T) {
	if got := newTestState().Init()(); got != (readAsked{}) {
		t.Errorf("Init gives %v", got)
	}
}

func TestStateScreenTakesAReport(t *testing.T) {
	s, cmd := feedState(t, newTestState(), StateRead{Sections: testSections()})
	if cmd != nil || s.reading || !s.reported || len(s.rows[packagesTab]) != 2 {
		t.Errorf("reading %v, reported %v, %d packages", s.reading, s.reported, len(s.rows[packagesTab]))
	}
}

func TestStateScreenAFailedReadKeepsTheRows(t *testing.T) {
	s, _ := feedState(t, reportedState(t), StateRead{Err: errors.New("no ssh\nat all")})
	if s.problem != "no ssh" || s.reading || len(s.rows[packagesTab]) != 2 {
		t.Fatalf("problem %q, reading %v, %d packages", s.problem, s.reading, len(s.rows[packagesTab]))
	}
	s, _ = feedState(t, s, StateRead{Sections: testSections()})
	if s.problem != "" {
		t.Errorf("the problem stays: %q", s.problem)
	}
}

func TestStateScreenKeepsAWarningOfAStepAlone(t *testing.T) {
	s, _ := feedState(t, newTestState(), Line{Said, "a line"}, Line{Warned, "a slow disk\nat last"})
	if s.warning != "a slow disk" {
		t.Errorf("warning %q", s.warning)
	}
}
