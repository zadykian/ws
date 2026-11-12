package tui

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/step"
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

// planned is how a scan ends a step: with changes, under a change of the step.
func planned(parent, name string, index int, changes ...step.Change) step.Event {
	return step.Event{Kind: step.Done, Parent: parent, Step: name, Index: index, Check: true, Changes: changes}
}

func TestAScanChoosesTheStepsWithChanges(t *testing.T) {
	l := chain().take(planned("base", "one", 0)).take(planned("base", "alone", 3, oneChange("a")))
	if want := []string{"[ ]", "[ ]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if l.rows[0].state != stepSame || l.rows[1].state != stepOpen || l.rows[3].state != stepChanged {
		t.Errorf("rows = %+v", l.rows)
	}
}

func TestAStepBehindChangesStartsChosen(t *testing.T) {
	behind := step.Event{Kind: step.Skipped, Parent: "base", Step: "two", Index: 1, After: "one"}
	l := chain().take(planned("base", "one", 0, oneChange("a"))).hear(Line{Said, "base: two: after one"}).take(behind)
	if want := []string{"[#]", "[x]", "[ ]", "[ ]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if len(l.rows[1].said) != 0 {
		t.Errorf("said = %+v, the list words that itself", l.rows[1].said)
	}
}

func TestAStepThatFailsStartsLeftOutWithItsError(t *testing.T) {
	failed := step.Event{Kind: step.Failed, Parent: "base", Step: "one", Err: errors.New("no disk")}
	row := chain().take(failed).rows[0]
	if row.on() || row.state != stepFailed || len(row.said) != 1 || row.said[0] != (Line{Warned, "no disk"}) {
		t.Errorf("row = %+v", row)
	}
}

func TestALineGoesToTheStepThatEndsPastIt(t *testing.T) {
	l := chain().hear(Line{Diffed, "base: one: writes a\n+b"}).hear(Line{Warned, "warning: old"})
	got := l.take(step.Event{Kind: step.Planned, Parent: "base", Step: "one"}).take(planned("base", "one", 0)).rows[0].said
	if want := []Line{{Diffed, "writes a\n+b"}, {Warned, "warning: old"}}; !slices.Equal(got, want) {
		t.Errorf("said = %+v, want %+v", got, want)
	}
}

func TestAStepOutsideTheListTakesItsLinesWithIt(t *testing.T) {
	l := chain().hear(Line{Said, "prepare: first: writes"}).take(planned("prepare", "first", 0))
	if got := l.take(planned("base", "one", 0)).rows[0].said; len(got) != 0 {
		t.Errorf("said = %+v, want none", got)
	}
}

// The run chooses by name, so a box left open beside its twin would run all the same.
func TestAStepThatComesTwiceIsOneChoice(t *testing.T) {
	l := newPickList([]Group{{Steps: []string{"ssh", "git", "ssh"}}})
	l = l.take(planned("", "ssh", 0, oneChange("a"))).take(planned("", "git", 1)).take(planned("", "ssh", 2))
	if want := []string{"[x]", "[ ]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
}

// threeRoles has changes in the first two.
func threeRoles() pickList {
	l := newPickList([]Group{{Name: "a", Steps: []string{"one"}}, {Name: "b", Steps: []string{"two", "three"}}, {Name: "c", Steps: []string{"four"}}})
	return l.take(planned("a", "one", 0, oneChange("x"))).take(planned("b", "two", 0, oneChange("x")))
}

func TestARoleIsNamedWhereItsChangesStayBehindAStepOn(t *testing.T) {
	for _, c := range []struct{ on, want []string }{
		{[]string{"four"}, []string{"a", "b"}},
		{[]string{"one"}, nil},
		{[]string{"one", "three"}, nil},
	} {
		if got := choosing(threeRoles(), c.on...).passedOver(); !slices.Equal(got, c.want) {
			t.Errorf("%q on: passed over = %q, want %q", c.on, got, c.want)
		}
	}
	flat := newPickList([]Group{{Steps: []string{"ssh", "git"}}}).take(planned("", "ssh", 0, oneChange("x")))
	if got := choosing(flat, "git").passedOver(); len(got) != 0 {
		t.Errorf("passed over = %q, want none in a flat list", got)
	}
}

func TestAWaitOfTheSameNameStaysWithItsStage(t *testing.T) {
	first := pickRow{stepRow: stepRow{name: "a"}, stage: "one"}
	second := pickRow{stepRow: stepRow{name: "a"}, stage: "two"}
	if first.sameChoice(second) {
		t.Errorf("%+v and %+v are one choice", first, second)
	}
}
