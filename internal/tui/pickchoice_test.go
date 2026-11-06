package tui

import (
	"slices"
	"testing"
)

// cursorOn returns l with the cursor on the step of that name.
func cursorOn(l pickList, name string) pickList {
	l.at = slices.IndexFunc(l.lines(), func(line pickLine) bool {
		return !line.heading && l.rows[line.row].name == name
	})
	return l
}

func TestATurnTicksAndClearsTheStepUnderTheCursor(t *testing.T) {
	l := cursorOn(twoRoles(), "Locale").toggled()
	if want := []string{"[ ]", "[x]", "[ ]", "[ ]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if got := l.toggled().only(); len(got) != 0 {
		t.Errorf("only = %q, want none", got)
	}
}

func TestALockedStepHoldsItsBox(t *testing.T) {
	l := cursorOn(choosing(chain(), "two"), "one")
	if got := l.toggled().only(); !slices.Equal(got, l.only()) {
		t.Errorf("only = %q, want %q", got, l.only())
	}
}

func TestATurnReachesAStepThatComesTwice(t *testing.T) {
	l := cursorOn(newPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}}), "ssh").toggled()
	if want := []string{"[x]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if want := []string{"[ ]", "[ ]", "[ ]"}; !slices.Equal(boxes(l.toggled()), want) {
		t.Errorf("boxes = %q, want %q", boxes(l.toggled()), want)
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
