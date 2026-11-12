package tui

import (
	"slices"
	"strings"
	"testing"
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
	if got, ok := l.current(l.at); !ok || got != (pickLine{3, true}) {
		t.Errorf("current = %v, %v", got, ok)
	}
}

func TestAFilterMatchesARoleOrAStepInAnyCase(t *testing.T) {
	if got, want := twoRoles().filtered("LOC").lines(), []pickLine{{1, true}, {1, false}}; !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
	if got, want := twoRoles().filtered("ssh").lines(), []pickLine{{3, true}, {3, false}}; !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
}

func TestAFilterKeepsTheCursorOnItsLine(t *testing.T) {
	l := twoRoles()
	l.at = 3
	l = l.filtered("t")
	got, ok := l.current(l.at)
	if want := (pickLine{2, false}); !ok || got != want {
		t.Errorf("current = %v, %v, want %v", got, ok, want)
	}
	l = l.filtered("ssh")
	if l.at != 0 {
		t.Errorf("at = %d, its line went, so the cursor starts the list", l.at)
	}
}

func TestAFilterCanLeaveNoLine(t *testing.T) {
	if got, ok := twoRoles().filtered("nothing here").current(0); ok {
		t.Errorf("current = %v, want none", got)
	}
}

func TestAViewDrawsARolesBoxBehindItsSteps(t *testing.T) {
	l := choosing(twoRoles(), "Packages", "Keys").take(planned("base", "Locale", 1))
	l.rows[1].chosen = false
	l.at = 1
	want := strings.Join([]string{
		"  [-] base",
		"▸   [x] · Packages",
		"    [ ] ✓ Locale",
		"    [ ] · Time",
		"  [x] ssh",
		"    [x] · Keys",
	}, "\n")
	if got := stripped(l.view(testTheme, 6)); got != want {
		t.Errorf("view:\n%s\nwant:\n%s", got, want)
	}
}

func TestAViewKeepsTheCursorInSight(t *testing.T) {
	l := newPickList([]Group{{Steps: strings.Fields("a b c d e f g h i j")}})
	l.at = 7
	got := strings.Split(stripped(l.view(testTheme, 3)), "\n")
	if len(got) != 3 || got[1] != "▸ [ ] · h" {
		t.Errorf("view = %q, want the cursor on the second of 3 lines", got)
	}
}

func TestAViewSaysThatAFilterLeftNothing(t *testing.T) {
	if got := stripped(twoRoles().filtered("nothing here").view(testTheme, 5)); got != "No step matches." {
		t.Errorf("view = %q", got)
	}
}
