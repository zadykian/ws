package list

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// cursorOn returns the cursor on the step of that name.
func cursorOn(l PickList, name string) widget.Cursor {
	return widget.Cursor(slices.IndexFunc(l.Lines(), func(line PickLine) bool {
		return !line.Heading && l.Rows[line.Row].Name == name
	}))
}

func TestATurnTicksAndClearsTheStepUnderTheCursor(t *testing.T) {
	l := twoRoles()
	at := cursorOn(l, "Locale")
	l = l.Toggled(at)
	if want := []string{"[ ]", "[x]", "[ ]", "[ ]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if got := l.Toggled(at).Only(); len(got) != 0 {
		t.Errorf("only = %q, want none", got)
	}
}

func TestALockedStepHoldsItsBox(t *testing.T) {
	l := choosing(chain(), "two")
	at := cursorOn(l, "one")
	if got := l.Toggled(at).Only(); !slices.Equal(got, l.Only()) {
		t.Errorf("only = %q, want %q", got, l.Only())
	}
}

func TestARolesBoxTurnsItsStepsOnThenOff(t *testing.T) {
	l := choosing(twoRoles(), "Locale", "Keys").Toggled(0)
	if want := []string{"[x]", "[x]", "[x]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if want := []string{"[ ]", "[ ]", "[ ]", "[x]"}; !slices.Equal(boxes(l.Toggled(0)), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.Toggled(0)), want)
	}
}

func TestARolesBoxTurnsTheStepsAFilterShows(t *testing.T) {
	l := twoRoles().Filtered("t") // Time alone, of base, and Keys has no t.
	if want := []string{"[ ]", "[ ]", "[x]", "[ ]"}; !slices.Equal(boxes(l.Toggled(0)), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.Toggled(0)), want)
	}
}

func TestATurnReachesAStepThatComesTwice(t *testing.T) {
	l := NewPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}}).Toggled(0)
	if want := []string{"[x]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if want := []string{"[ ]", "[ ]", "[ ]"}; !slices.Equal(boxes(l.Toggled(0)), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.Toggled(0)), want)
	}
}

func TestABulkChoiceHoldsToWhatAFilterShows(t *testing.T) {
	l := choosing(twoRoles(), "Keys").Filtered("base").Chose(func(PickRow) bool { return true })
	if want := []string{"[x]", "[x]", "[x]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	l = l.Chose(func(PickRow) bool { return false })
	if want := []string{"[ ]", "[ ]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
}

func TestCountTakesTheStepsItIsGiven(t *testing.T) {
	on, all := choosing(twoRoles(), "Time", "Keys").count(func(r PickRow) bool { return r.Parent == "base" })
	if on != 1 || all != 3 {
		t.Errorf("count = %d of %d, want 1 of 3", on, all)
	}
}
