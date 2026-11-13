package list

import (
	"slices"
	"strings"
	"testing"
)

// twoRoles is the list of two roles, of three steps and of one.
func twoRoles() PickList {
	return NewPickList([]Group{{Name: "base", Steps: []string{"Packages", "Locale", "Time"}}, {Name: "ssh", Steps: []string{"Keys"}}})
}

func TestEachRoleOpensUnderItsHeading(t *testing.T) {
	got := twoRoles().Lines()
	want := []PickLine{{0, true}, {0, false}, {1, false}, {2, false}, {3, true}, {3, false}}
	if !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
}

func TestAFlatListHasNoHeading(t *testing.T) {
	got := NewPickList([]Group{{Steps: []string{"ssh", "git"}}}).Lines()
	if want := []PickLine{{0, false}, {1, false}}; !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
}

func TestCurrentIsTheLineUnderTheCursor(t *testing.T) {
	if got, ok := twoRoles().Current(4); !ok || got != (PickLine{3, true}) {
		t.Errorf("current = %v, %v", got, ok)
	}
}

func TestAFilterMatchesARoleOrAStepInAnyCase(t *testing.T) {
	if got, want := twoRoles().Filtered("LOC").Lines(), []PickLine{{1, true}, {1, false}}; !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
	if got, want := twoRoles().Filtered("ssh").Lines(), []PickLine{{3, true}, {3, false}}; !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
}

func TestAFilterKeepsTheCursorOnItsLine(t *testing.T) {
	l, at := twoRoles().Narrowed(3, "t")
	got, ok := l.Current(at)
	if want := (PickLine{2, false}); !ok || got != want {
		t.Errorf("current = %v, %v, want %v", got, ok, want)
	}
	if _, at = l.Narrowed(at, "ssh"); at != 0 {
		t.Errorf("at = %d, its line went, so the cursor starts the list", at)
	}
}

func TestAFilterCanLeaveNoLine(t *testing.T) {
	if got, ok := twoRoles().Filtered("nothing here").Current(0); ok {
		t.Errorf("current = %v, want none", got)
	}
}

func TestAViewDrawsARolesBoxBehindItsSteps(t *testing.T) {
	l := choosing(twoRoles(), "Packages", "Keys").Take(planned("base", "Locale", 1))
	l.Rows[1].Chosen = false
	want := strings.Join([]string{
		"  [-] base",
		"▸   [x] · Packages",
		"    [ ] ✓ Locale",
		"    [ ] · Time",
		"  [x] ssh",
		"    [x] · Keys",
	}, "\n")
	if got := stripped(l.View(testTheme, 6, 1)); got != want {
		t.Errorf("view:\n%s\nwant:\n%s", got, want)
	}
}

func TestAViewKeepsTheCursorInSight(t *testing.T) {
	l := NewPickList([]Group{{Steps: strings.Fields("a b c d e f g h i j")}})
	got := strings.Split(stripped(l.View(testTheme, 3, 7)), "\n")
	if len(got) != 3 || got[1] != "▸ [ ] · h" {
		t.Errorf("view = %q, want the cursor on the second of 3 lines", got)
	}
}

func TestAViewSaysThatAFilterLeftNothing(t *testing.T) {
	if got := stripped(twoRoles().Filtered("nothing here").View(testTheme, 5, 0)); got != "No step matches." {
		t.Errorf("view = %q", got)
	}
}
