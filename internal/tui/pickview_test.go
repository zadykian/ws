package tui

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// twoRoles is the list of two roles, of three steps and of one.
func twoRoles() pickList {
	return newPickList([]Group{{Name: "base", Steps: []string{"Packages", "Locale", "Time"}}, {Name: "ssh", Steps: []string{"Keys"}}})
}

func TestEachRoleOpensUnderItsHeading(t *testing.T) {
	got := twoRoles().lines()
	want := []pickLine{{0, true}, {0, false}, {1, false}, {2, false}, {3, true}, {3, false}}
	if !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
}

func TestAFlatListHasNoHeading(t *testing.T) {
	got := newPickList([]Group{{Steps: []string{"ssh", "git"}}}).lines()
	if want := []pickLine{{0, false}, {1, false}}; !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
}

func TestCurrentIsTheLineUnderTheCursor(t *testing.T) {
	l := twoRoles()
	l.at = 4
	if got, ok := l.current(); !ok || got != (pickLine{3, true}) {
		t.Errorf("current = %v, %v", got, ok)
	}
}

func TestTheCursorStopsAtTheEnds(t *testing.T) {
	cases := []struct {
		msg  tea.KeyPressMsg
		from int
		want int
	}{
		{press('k'), 0, 0},
		{press('j'), 0, 1},
		{press('j'), 5, 5},
		{press('G'), 0, 5},
		{press('g'), 3, 0},
		{tea.KeyPressMsg{Code: tea.KeyPgDown}, 1, 5},
		{tea.KeyPressMsg{Code: tea.KeyPgUp}, 5, 1},
	}
	for _, c := range cases {
		l := twoRoles()
		l.at = c.from
		if got := l.moved(c.msg, 4).at; got != c.want {
			t.Errorf("%v from %d = %d, want %d", c.msg, c.from, got, c.want)
		}
	}
}
