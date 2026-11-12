package tui

import (
	"slices"
	"testing"
)

// cursorOn returns l with the cursor on the step of that name.
func cursorOn(l pickList, name string) pickList {
	l.at = cursor(slices.IndexFunc(l.lines(), func(line pickLine) bool {
		return !line.heading && l.rows[line.row].name == name
	}))
	return l
}

func TestATurnTicksAndClearsTheStepUnderTheCursor(t *testing.T) {
	l := cursorOn(twoRoles(), "Locale").toggle()
	if want := []string{"[ ]", "[x]", "[ ]", "[ ]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if got := l.toggle().only(); len(got) != 0 {
		t.Errorf("only = %q, want none", got)
	}
}

func TestALockedStepHoldsItsBox(t *testing.T) {
	l := cursorOn(choosing(chain(), "two"), "one")
	if got := l.toggle().only(); !slices.Equal(got, l.only()) {
		t.Errorf("only = %q, want %q", got, l.only())
	}
}

func TestARolesBoxTurnsItsStepsOnThenOff(t *testing.T) {
	l := choosing(twoRoles(), "Locale", "Keys")
	l.at = 0
	l = l.toggle()
	if want := []string{"[x]", "[x]", "[x]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if want := []string{"[ ]", "[ ]", "[ ]", "[x]"}; !slices.Equal(boxes(l.toggle()), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.toggle()), want)
	}
}

func TestARolesBoxTurnsTheStepsAFilterShows(t *testing.T) {
	l := twoRoles().filtered("t") // Time alone, of base, and Keys has no t.
	l.at = 0
	if want := []string{"[ ]", "[ ]", "[x]", "[ ]"}; !slices.Equal(boxes(l.toggle()), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.toggle()), want)
	}
}

func TestATurnReachesAStepThatComesTwice(t *testing.T) {
	l := cursorOn(newPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}}), "ssh").toggle()
	if want := []string{"[x]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if want := []string{"[ ]", "[ ]", "[ ]"}; !slices.Equal(boxes(l.toggle()), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.toggle()), want)
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

// toggle turns the box under the cursor the list holds.
func (l pickList) toggle() pickList { return l.toggled(l.at) }
