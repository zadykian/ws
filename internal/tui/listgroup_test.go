package tui

import (
	"strings"
	"testing"
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
