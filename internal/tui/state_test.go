package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The messages of a test's read and leave.
type (
	readAsked struct{}
	leftAsked struct{}
)

var (
	sized = tea.WindowSizeMsg{Width: 80, Height: 24}
)

func newTestState() StateScreen {
	read := func() tea.Msg { return readAsked{} }
	leave := func() tea.Msg { return leftAsked{} }
	return NewStateScreen(testTheme, "ws server state", read, leave)
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

func TestStateScreenButtons(t *testing.T) {
	cases := []struct {
		name string
		keys []tea.Msg
		want tea.Msg
	}{
		{"r reads again", []tea.Msg{press('r')}, readAsked{}},
		{"q leaves", []tea.Msg{press('q')}, leftAsked{}},
		{"Enter presses the button in focus", []tea.Msg{shiftTab, enter}, leftAsked{}},
	}
	for _, c := range cases {
		_, cmd := feedState(t, reportedState(t), c.keys...)
		if cmd == nil || cmd() != c.want {
			t.Errorf("%s: no %T", c.name, c.want)
		}
	}
}

func TestStateScreenReadsOnceAtATime(t *testing.T) {
	if _, cmd := feedState(t, newTestState(), press('r')); cmd != nil {
		t.Error("r asks during the first read")
	}
	s, _ := feedState(t, reportedState(t), Line{Warned, "a slow disk"}, press('r'))
	if !s.reading || s.warning != "" {
		t.Fatalf("after r: reading %v, warning %q", s.reading, s.warning)
	}
	if _, cmd := feedState(t, s, press('r')); cmd != nil {
		t.Error("a second r reads beside the first")
	}
	if s, _ = feedState(t, s, StateRead{}); s.reading || len(s.rows[packagesTab]) != 0 {
		t.Errorf("a new report: reading %v, %d packages", s.reading, len(s.rows[packagesTab]))
	}
	if _, cmd := feedState(t, s, press('r')); cmd == nil {
		t.Error("r can't read again")
	}
}

func TestStateScreenOtherKeysAskForNothing(t *testing.T) {
	for _, msg := range []tea.Msg{press('j'), enter, esc, tab, shiftTab, press('x'), press('G'), press('h')} {
		if _, cmd := feedState(t, reportedState(t), msg); cmd != nil {
			t.Errorf("%v gives a Cmd", msg)
		}
	}
}

func TestStateScreenTabWalksTheTabsThenTheButtons(t *testing.T) {
	type place struct{ tab, button int }
	s := reportedState(t)
	for _, want := range []place{{1, -1}, {2, -1}, {3, -1}, {4, -1}, {4, 0}, {4, 1}, {0, -1}} {
		s, _ = feedState(t, s, tab)
		if got := (place{s.tab, s.footer.Focused()}); got != want {
			t.Fatalf("Tab: %+v, want %+v", got, want)
		}
	}
	for _, want := range []place{{0, 1}, {0, 0}, {4, -1}, {3, -1}} {
		s, _ = feedState(t, s, shiftTab)
		if got := (place{s.tab, s.footer.Focused()}); got != want {
			t.Fatalf("Shift-Tab: %+v, want %+v", got, want)
		}
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
