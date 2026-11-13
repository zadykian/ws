package tui

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// chain is a role whose steps wait on each other, beside one that waits on nothing.
func chain() PickList {
	waits := map[string]string{"two": "one", "three": "two"}
	return NewPickList([]Group{{Name: "base", Steps: []string{"one", "two", "three", "alone"}, Waits: waits}})
}

// choosing returns l with the named steps' own boxes ticked.
func choosing(l PickList, names ...string) PickList {
	l.Rows = slices.Clone(l.Rows)
	for i := range l.Rows {
		l.Rows[i].Chosen = slices.Contains(names, l.Rows[i].Name)
	}
	return l.Settled()
}

// boxes draws each row's box in order, as the eye reads a list.
func boxes(l PickList) []string {
	drawn := make([]string, len(l.Rows))
	for i, r := range l.Rows {
		drawn[i] = r.Box()
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
	l := NewPickList([]Group{{Name: "base", Steps: []string{"one", "two", "three"}, Waits: waits}})
	got := choosing(l, "two", "three").Rows[0].LockedBy
	if want := []string{"three", "two"}; !slices.Equal(got, want) {
		t.Errorf("locked by %q, want %q", got, want)
	}
}

func TestALockStaysInsideItsGroup(t *testing.T) {
	l := NewPickList([]Group{
		{Name: "a", Steps: []string{"one", "two"}},
		{Name: "b", Steps: []string{"one", "two"}, Waits: map[string]string{"two": "one"}},
	})
	if got, want := boxes(choosing(l, "two")), []string{"[ ]", "[x]", "[#]", "[x]"}; !slices.Equal(got, want) {
		t.Errorf("boxes = %q, want %q", got, want)
	}
}

func TestOnlyNamesTheStepsOnEachOnce(t *testing.T) {
	if got := chain().Only(); len(got) != 0 {
		t.Errorf("only = %q, want none", got)
	}
	if got, want := choosing(chain(), "two").Only(), []string{"base/one", "base/two"}; !slices.Equal(got, want) {
		t.Errorf("only = %q, want %q", got, want)
	}
	flat := choosing(NewPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}}), "ssh")
	if got, want := flat.Only(), []string{"ssh"}; !slices.Equal(got, want) {
		t.Errorf("only = %q, want %q", got, want)
	}
}

func TestABoxSaysWhetherItsStepIsChosen(t *testing.T) {
	rows := choosing(chain(), "two").Rows
	if rows[0].Box() != "[#]" || rows[1].Box() != "[x]" || rows[2].Box() != "[ ]" {
		t.Errorf("boxes = %q, %q, %q", rows[0].Box(), rows[1].Box(), rows[2].Box())
	}
}

func TestAScanChoosesTheStepsWithChanges(t *testing.T) {
	l := chain().Take(planned("base", "one", 0)).Take(planned("base", "alone", 3, oneChange("a")))
	if want := []string{"[ ]", "[ ]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if l.Rows[0].State != StepSame || l.Rows[1].State != StepOpen || l.Rows[3].State != StepChanged {
		t.Errorf("rows = %+v", l.Rows)
	}
}

func TestAStepBehindChangesStartsChosen(t *testing.T) {
	behind := step.Event{Kind: step.Skipped, Parent: "base", Step: "two", Index: 1, After: "one"}
	l := chain().Take(planned("base", "one", 0, oneChange("a"))).Hear(widget.Line{Kind: widget.Said, Text: "base: two: after one"}).Take(behind)
	if want := []string{"[#]", "[x]", "[ ]", "[ ]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if len(l.Rows[1].Said) != 0 {
		t.Errorf("said = %+v, the list words that itself", l.Rows[1].Said)
	}
}

func TestAStepThatFailsStartsLeftOutWithItsError(t *testing.T) {
	failed := step.Event{Kind: step.Failed, Parent: "base", Step: "one", Err: errors.New("no disk")}
	row := chain().Take(failed).Rows[0]
	if row.On() || row.State != StepFailed || len(row.Said) != 1 || row.Said[0] != (widget.Line{Kind: widget.Warned, Text: "no disk"}) {
		t.Errorf("row = %+v", row)
	}
}

func TestALineGoesToTheStepThatEndsPastIt(t *testing.T) {
	l := chain().Hear(widget.Line{Kind: widget.Diffed, Text: "base: one: writes a\n+b"}).Hear(widget.Line{Kind: widget.Warned, Text: "warning: old"})
	got := l.Take(step.Event{Kind: step.Planned, Parent: "base", Step: "one"}).Take(planned("base", "one", 0)).Rows[0].Said
	if want := []widget.Line{{Kind: widget.Diffed, Text: "writes a\n+b"}, {Kind: widget.Warned, Text: "warning: old"}}; !slices.Equal(got, want) {
		t.Errorf("said = %+v, want %+v", got, want)
	}
}

func TestAStepOutsideTheListTakesItsLinesWithIt(t *testing.T) {
	l := chain().Hear(widget.Line{Kind: widget.Said, Text: "prepare: first: writes"}).Take(planned("prepare", "first", 0))
	if got := l.Take(planned("base", "one", 0)).Rows[0].Said; len(got) != 0 {
		t.Errorf("said = %+v, want none", got)
	}
}

// The run chooses by name, so a box left open beside its twin would run all the same.
func TestAStepThatComesTwiceIsOneChoice(t *testing.T) {
	l := NewPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}})
	l = l.Take(planned("", "ssh", 0, oneChange("a"))).Take(planned("", "git", 1)).Take(planned("", "ssh", 2))
	if want := []string{"[x]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
}

// threeRoles has changes in the first two.
func threeRoles() PickList {
	l := NewPickList([]Group{{Name: "a", Steps: []string{"one"}}, {Name: "b", Steps: []string{"two", "three"}}, {Name: "c", Steps: []string{"four"}}})
	return l.Take(planned("a", "one", 0, oneChange("x"))).Take(planned("b", "two", 0, oneChange("x")))
}

func TestARoleIsNamedWhereItsChangesStayBehindAStepOn(t *testing.T) {
	for _, c := range []struct{ on, want []string }{
		{[]string{"four"}, []string{"a", "b"}},
		{[]string{"one"}, nil},
		{[]string{"one", "three"}, nil},
	} {
		if got := choosing(threeRoles(), c.on...).PassedOver(); !slices.Equal(got, c.want) {
			t.Errorf("%q on: passed over = %q, want %q", c.on, got, c.want)
		}
	}
	flat := NewPickList([]Group{{Steps: []string{"ssh", "git"}}}).Take(planned("", "ssh", 0, oneChange("x")))
	if got := choosing(flat, "git").PassedOver(); len(got) != 0 {
		t.Errorf("passed over = %q, want none in a flat list", got)
	}
}

func TestARowTakesWhatTheScanSaidOfItsStep(t *testing.T) {
	row := PickRow{StepRow: StepRow{Name: "one"}}
	said := []widget.Line{{Kind: widget.Said, Text: "one: sets it"}}
	got := row.Took(planned("", "one", 0, oneChange("a")), said)
	if !got.Chosen || got.State != StepChanged || len(got.Said) != 1 || got.Said[0].Text != "sets it" {
		t.Errorf("row = %+v", got)
	}
	failed := row.Took(step.Event{Kind: step.Failed, Err: errors.New("no")}, said)
	if failed.Chosen || failed.Said[len(failed.Said)-1].Kind != widget.Warned {
		t.Errorf("row = %+v", failed)
	}
}
