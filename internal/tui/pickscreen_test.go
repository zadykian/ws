package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/step"
)

// pickSent returns s past msgs.
func pickSent(s pickScreen, msgs ...tea.Msg) pickScreen {
	for _, msg := range msgs {
		model, _ := s.Update(msg)
		s = model.(pickScreen)
	}
	return s
}

// pickShown is the words of s's frame.
func pickShown(s pickScreen) string { return stripped(s.View().Content) }

// testPick is a screen at 80 by 24, its scan under way.
func testPick() pickScreen {
	groups := []Group{{Name: "base", Steps: []string{"Packages", "Locale"}}, {Name: "ssh", Steps: []string{"Keys"}}}
	return pickSent(newPickScreen("ws server apply", testTheme, groups), tea.WindowSizeMsg{Width: 80, Height: 24})
}

func TestPickScreenShowsEveryStepBeforeTheScan(t *testing.T) {
	got := pickShown(testPick())
	for _, want := range []string{"ws server apply", "scanning 0/3", "[ ] base", "[ ] · Packages", "[ ] ssh", "[ ] · Keys"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
}

func TestPickScreenFollowsTheScan(t *testing.T) {
	s := pickSent(testPick(), Line{Said, "base: Packages: installs vim"}, step.Event{Kind: step.Planned, Parent: "base", Step: "Packages"},
		planned("base", "Packages", 0, oneChange("installs vim")))
	if got := pickShown(s); !strings.Contains(got, "scanning 1/3") || !strings.Contains(got, "[x] ● Packages") {
		t.Errorf("frame:\n%s", got)
	}
	if got := pickShown(pickSent(s, scannedMsg{})); !strings.Contains(got, "1 of 3 chosen") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestPickScreenMovesDuringTheScan(t *testing.T) {
	if got := pickSent(testPick(), press('j'), press('j')).list.at; got != 2 {
		t.Errorf("at = %d, want 2", got)
	}
}

func TestPickScreenFitsAnyWindow(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 40, Height: 12}, {Width: 30, Height: 8}, {Width: 12, Height: 5}} {
		lines := strings.Split(pickShown(pickSent(testPick(), size)), "\n")
		if len(lines) > size.Height {
			t.Errorf("%v: %d lines", size, len(lines))
		}
	}
}
