package tui

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/step"
)

func TestARunMovesTheMarksAndLeavesTheBoxes(t *testing.T) {
	l := choosing(newQueue(testStages()), "Tools")
	l, line := l.follow(step.Event{Kind: step.Started, Step: "Sudo", Index: 2})
	if l.side != WsSide || line != 0 {
		t.Errorf("side %d, line %d", l.side, line)
	}
	l, line = l.follow(step.Event{Kind: step.Done, Step: "Tools", Index: 0, LeftOut: true})
	if line >= 0 {
		t.Errorf("line %d: a step done takes no cursor", line)
	}
	if got := drawn(l, MacSide)[0]; got != " 1 [x] ○ Tools" {
		t.Errorf("row = %q", got)
	}
}

func TestARunLearnsTheStepsOfARoleAndOpensOneAgain(t *testing.T) {
	event := step.Event{Kind: step.Planned, Parent: "ssh", Step: "Keys"}
	l, _ := newQueue(testStages()).follow(event)
	if got := drawn(l, WsSide)[3:]; !slices.Equal(got, []string{"     [ ] ssh", "       [ ] · Keys"}) {
		t.Errorf("list = %q", got)
	}
	event.Kind = step.Done
	if got := followed(l, event).rows[l.place(event)].state; got != stepSame {
		t.Errorf("state = %d", got)
	}
	event.Kind = step.Planned
	if got := followed(l, event).rows[l.place(event)].state; got != stepOpen {
		t.Errorf("state = %d", got)
	}
}

// followed is l after e, the cursor aside.
func followed(l pickList, e step.Event) pickList {
	l, _ = l.follow(e)
	return l
}
