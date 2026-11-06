package tui

import (
	"slices"
	"testing"
)

// chain is a role whose steps wait on each other, beside one that waits on nothing.
func chain() pickList {
	waits := map[string]string{"two": "one", "three": "two"}
	return newPickList([]Group{{Name: "base", Steps: []string{"one", "two", "three", "alone"}, Waits: waits}})
}

// choosing returns l with the named steps' own boxes ticked.
func choosing(l pickList, names ...string) pickList {
	l.rows = slices.Clone(l.rows)
	for i := range l.rows {
		l.rows[i].chosen = slices.Contains(names, l.rows[i].name)
	}
	return l.settled()
}

// boxes draws each row's box in order, as the eye reads a list.
func boxes(l pickList) []string {
	drawn := make([]string, len(l.rows))
	for i, r := range l.rows {
		drawn[i] = r.box()
	}
	return drawn
}

func TestAChosenStepLocksWhatItWaitsOnDownItsChain(t *testing.T) {
	got := boxes(choosing(chain(), "three"))
	if want := []string{"[#]", "[#]", "[x]", "[ ]"}; !slices.Equal(got, want) {
		t.Errorf("boxes = %q, want %q", got, want)
	}
}

func TestALockGoesWithTheStepThatHeldIt(t *testing.T) {
	l := choosing(choosing(chain(), "three"), "alone")
	if want := []string{"[ ]", "[ ]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
}

func TestALockNamesEveryStepThatHoldsIt(t *testing.T) {
	waits := map[string]string{"two": "one", "three": "one"}
	l := newPickList([]Group{{Name: "base", Steps: []string{"one", "two", "three"}, Waits: waits}})
	got := choosing(l, "two", "three").rows[0].lockedBy
	if want := []string{"three", "two"}; !slices.Equal(got, want) {
		t.Errorf("locked by %q, want %q", got, want)
	}
}

func TestALockStaysInsideItsGroup(t *testing.T) {
	l := newPickList([]Group{
		{Name: "a", Steps: []string{"one", "two"}},
		{Name: "b", Steps: []string{"one", "two"}, Waits: map[string]string{"two": "one"}},
	})
	if got, want := boxes(choosing(l, "two")), []string{"[ ]", "[x]", "[#]", "[x]"}; !slices.Equal(got, want) {
		t.Errorf("boxes = %q, want %q", got, want)
	}
}

func TestOnlyNamesTheStepsOnEachOnce(t *testing.T) {
	if got := chain().only(); len(got) != 0 {
		t.Errorf("only = %q, want none", got)
	}
	if got, want := choosing(chain(), "two").only(), []string{"base/one", "base/two"}; !slices.Equal(got, want) {
		t.Errorf("only = %q, want %q", got, want)
	}
	flat := choosing(newPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}}), "ssh")
	if got, want := flat.only(), []string{"ssh"}; !slices.Equal(got, want) {
		t.Errorf("only = %q, want %q", got, want)
	}
}

func TestABoxSaysWhetherItsStepIsChosen(t *testing.T) {
	rows := choosing(chain(), "two").rows
	if rows[0].box() != "[#]" || rows[1].box() != "[x]" || rows[2].box() != "[ ]" {
		t.Errorf("boxes = %q, %q, %q", rows[0].box(), rows[1].box(), rows[2].box())
	}
}
