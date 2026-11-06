package tui

import (
	"slices"
	"testing"
)

// chain is a role of a few steps.
func chain() pickList {
	return newPickList([]Group{{Name: "base", Steps: []string{"one", "two", "three", "alone"}}})
}

// choosing returns l with the named steps' own boxes ticked.
func choosing(l pickList, names ...string) pickList {
	l.rows = slices.Clone(l.rows)
	for i := range l.rows {
		l.rows[i].chosen = slices.Contains(names, l.rows[i].name)
	}
	return l
}

func TestOnlyNamesTheStepsOnEachOnce(t *testing.T) {
	if got := chain().only(); len(got) != 0 {
		t.Errorf("only = %q, want none", got)
	}
	if got, want := choosing(chain(), "two").only(), []string{"base/two"}; !slices.Equal(got, want) {
		t.Errorf("only = %q, want %q", got, want)
	}
	flat := choosing(newPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}}), "ssh")
	if got, want := flat.only(), []string{"ssh"}; !slices.Equal(got, want) {
		t.Errorf("only = %q, want %q", got, want)
	}
}

func TestABoxSaysWhetherItsStepIsChosen(t *testing.T) {
	rows := choosing(chain(), "two").rows
	if rows[0].box() != "[ ]" || rows[1].box() != "[x]" {
		t.Errorf("boxes = %q, %q", rows[0].box(), rows[1].box())
	}
}
