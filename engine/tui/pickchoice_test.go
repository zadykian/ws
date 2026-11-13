package tui

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// cursorOn returns the cursor on the step of that name.
func cursorOn(l pickList, name string) widget.Cursor {
	return widget.Cursor(slices.IndexFunc(l.lines(), func(line pickLine) bool {
		return !line.heading && l.rows[line.row].name == name
	}))
}

func TestATurnTicksAndClearsTheStepUnderTheCursor(t *testing.T) {
	l := twoRoles()
	at := cursorOn(l, "Locale")
	l = l.toggled(at)
	if want := []string{"[ ]", "[x]", "[ ]", "[ ]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if got := l.toggled(at).only(); len(got) != 0 {
		t.Errorf("only = %q, want none", got)
	}
}

func TestALockedStepHoldsItsBox(t *testing.T) {
	l := choosing(chain(), "two")
	at := cursorOn(l, "one")
	if got := l.toggled(at).only(); !slices.Equal(got, l.only()) {
		t.Errorf("only = %q, want %q", got, l.only())
	}
}

func TestARolesBoxTurnsItsStepsOnThenOff(t *testing.T) {
	l := choosing(twoRoles(), "Locale", "Keys").toggled(0)
	if want := []string{"[x]", "[x]", "[x]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if want := []string{"[ ]", "[ ]", "[ ]", "[x]"}; !slices.Equal(boxes(l.toggled(0)), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.toggled(0)), want)
	}
}

func TestARolesBoxTurnsTheStepsAFilterShows(t *testing.T) {
	l := twoRoles().filtered("t") // Time alone, of base, and Keys has no t.
	if want := []string{"[ ]", "[ ]", "[x]", "[ ]"}; !slices.Equal(boxes(l.toggled(0)), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.toggled(0)), want)
	}
}

func TestATurnReachesAStepThatComesTwice(t *testing.T) {
	l := newPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}}).toggled(0)
	if want := []string{"[x]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if want := []string{"[ ]", "[ ]", "[ ]"}; !slices.Equal(boxes(l.toggled(0)), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.toggled(0)), want)
	}
}

func TestABulkChoiceHoldsToWhatAFilterShows(t *testing.T) {
	l := choosing(twoRoles(), "Keys").filtered("base").chose(func(pickRow) bool { return true })
	if want := []string{"[x]", "[x]", "[x]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	l = l.chose(func(pickRow) bool { return false })
	if want := []string{"[ ]", "[ ]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
}

func TestCountTakesTheStepsItIsGiven(t *testing.T) {
	on, all := choosing(twoRoles(), "Time", "Keys").count(func(r pickRow) bool { return r.parent == "base" })
	if on != 1 || all != 3 {
		t.Errorf("count = %d of %d, want 1 of 3", on, all)
	}
}
