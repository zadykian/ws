package tui

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

func formed() pickList {
	return newFormedList([]listGroup{
		{Group: Group{Name: "Setup", Steps: []string{"one", "two"}}, form: groupBare},
		{Group: Group{Name: "base", Steps: []string{"Packages"}}},
		{Group: Group{Name: "ssh"}, form: groupUnread},
	})
}

func drawnList(l pickList) string {
	var got []string
	for _, line := range l.lines() {
		got = append(got, stripped(l.render(testTheme, line)))
	}
	return strings.Join(got, "|")
}

func TestABareGroupHasNoHeading(t *testing.T) {
	if got, want := drawnList(formed()), "[ ] · one|[ ] · two|[ ] base|  [ ] · Packages|[ ] · ssh"; got != want {
		t.Errorf("list = %q, want %q", got, want)
	}
}

func TestAFilterThatMissesABareGroupsNameBringsItsHeadingBack(t *testing.T) {
	if got, want := drawnList(formed().filtered("two")), "[ ] Setup|[ ] · two"; got != want {
		t.Errorf("list = %q, want %q", got, want)
	}
}

func TestAStepTheScanLackedJoinsItsGroup(t *testing.T) {
	for _, tc := range []struct {
		name, parent string
		want         string
	}{
		{"three", "Setup", "[ ] · one|[ ] · two|[ ] · three|[ ] base|  [ ] · Packages|[ ] · ssh"},
		{"Locale", "base", "[ ] · one|[ ] · two|[ ] base|  [ ] · Packages|  [ ] · Locale|[ ] · ssh"},
		{"Keys", "ssh", "[ ] · one|[ ] · two|[ ] base|  [ ] · Packages|[ ] ssh|  [ ] · Keys"},
		{"Run", "Run", "[ ] · Run|[ ] · one|[ ] · two|[ ] base|  [ ] · Packages|[ ] · ssh"},
	} {
		e := step.Event{Kind: step.Planned, Parent: tc.parent, Step: tc.name, Index: 9}
		if got := drawnList(formed().joined(e)); got != tc.want {
			t.Errorf("%s joins to %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestAStepTheListHasJoinsNothing(t *testing.T) {
	for _, e := range []step.Event{{Parent: "base", Step: "Packages", Index: 0}, {Step: "Setup", Index: 1}} {
		if got := formed().joined(e); len(got.rows) != len(formed().rows) {
			t.Errorf("%+v joins %+v", e, got.rows)
		}
	}
}
