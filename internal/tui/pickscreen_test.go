package tui

import (
	"slices"
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
	return pickSent(newPickScreen("ws server apply", testTheme, groups, make(chan Choice, 1)), tea.WindowSizeMsg{Width: 80, Height: 24})
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
		for _, keys := range [][]tea.Msg{nil, {press('/'), press('s')}} {
			lines := strings.Split(pickShown(pickSent(testPick(), append(keys, size)...)), "\n")
			if len(lines) > size.Height {
				t.Errorf("%v: %d lines", size, len(lines))
			}
		}
	}
}

// scannedPick is a screen past its scan, which found changes in Packages and in Keys.
func scannedPick() pickScreen {
	return pickSent(testPick(), planned("base", "Packages", 0, oneChange("a")), planned("base", "Locale", 1),
		planned("ssh", "Keys", 0, oneChange("a")), scannedMsg{})
}

func TestPickScreenKeysTurnTheBoxes(t *testing.T) {
	space := tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	cases := []struct {
		name string
		keys []tea.Msg
		want []string
	}{
		{"a step", []tea.Msg{press('j'), space}, []string{"ssh/Keys"}},
		{"a role", []tea.Msg{space}, []string{"base/Packages", "base/Locale", "ssh/Keys"}},
		{"all", []tea.Msg{press('a')}, []string{"base/Packages", "base/Locale", "ssh/Keys"}},
		{"none", []tea.Msg{press('n')}, nil},
		{"invert", []tea.Msg{press('i')}, []string{"base/Locale"}},
		{"pending", []tea.Msg{press('a'), press('p')}, []string{"base/Packages", "ssh/Keys"}},
	}
	for _, c := range cases {
		if got := pickSent(scannedPick(), c.keys...).list.only(); !slices.Equal(got, c.want) {
			t.Errorf("%s: only = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPickScreenHoldsItsBoxesDuringTheScan(t *testing.T) {
	if got := pickSent(testPick(), press('a')).list.only(); len(got) != 0 {
		t.Errorf("only = %q, want none", got)
	}
}

// picked presses msg on s and returns the choice that it gives, with false where it gives none.
func picked(s pickScreen, msg tea.Msg) (Choice, bool) {
	picks := make(chan Choice, 1)
	s.picks = picks
	_, leaving := s.Update(msg)
	if leaving == nil {
		return Choice{}, false
	}
	leaving()
	return <-picks, true
}

func TestPickScreenGivesItsChoice(t *testing.T) {
	cases := []struct {
		msg  tea.Msg
		want Choice
	}{
		{enter, Choice{Only: []string{"base/Packages", "ssh/Keys"}}},
		{press('c'), Choice{Only: []string{"base/Packages", "ssh/Keys"}, Check: true}},
		{press('q'), Choice{}},
	}
	for _, c := range cases {
		got, ok := picked(scannedPick(), c.msg)
		if !ok || !slices.Equal(got.Only, c.want.Only) || got.Check != c.want.Check {
			t.Errorf("%v: choice = %+v, %v, want %+v", c.msg, got, ok, c.want)
		}
	}
}

// An empty Only means every step, so nothing chosen must never reach a run.
func TestPickScreenGivesNoRunWhereItHoldsNone(t *testing.T) {
	for name, s := range map[string]pickScreen{
		"nothing chosen": pickSent(scannedPick(), press('n')),
		"the scan":       testPick(),
		"past q":         pickSent(scannedPick(), press('q')),
	} {
		for _, msg := range []tea.Msg{enter, press('c')} {
			if got, ok := picked(s, msg); ok {
				t.Errorf("%s: %v gave %+v", name, msg, got)
			}
		}
	}
	if _, ok := picked(testPick(), press('q')); !ok {
		t.Error("q gave no choice during the scan")
	}
}

func TestPickScreenFilterTakesTheKeysUntilEnter(t *testing.T) {
	s := pickSent(testPick(), press('/'), press('k'), press('e'), press('q'), tea.KeyPressMsg{Code: tea.KeyBackspace})
	if got := pickShown(s); !strings.Contains(got, "/ ke") || !strings.Contains(got, "Keys") || strings.Contains(got, "Packages") {
		t.Errorf("frame:\n%s", got)
	}
	s = pickSent(s, enter, press('j'))
	if got := s.list.at; s.editing || got != 1 {
		t.Errorf("editing = %v, at = %d: keys past Enter move the list", s.editing, got)
	}
}

func TestPickScreenEscDropsTheFilter(t *testing.T) {
	for _, keys := range [][]tea.Msg{{press('/'), press('k'), esc}, {press('/'), press('k'), enter, esc}} {
		if s := pickSent(testPick(), keys...); s.filtering() {
			t.Errorf("%v: words = %q, editing = %v", keys, s.list.words, s.editing)
		}
	}
}
