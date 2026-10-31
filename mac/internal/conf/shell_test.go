package conf

import (
	"strings"
	"testing"
)

func TestAssignments(t *testing.T) {
	got := Conf{User: "root", Target: "lan", GitName: "O'Neil $(x)"}.Assignments()
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != len(keys) {
		t.Fatalf("%d lines, want %d: %q", len(lines), len(keys), got)
	}
	for _, want := range []string{"WS_USER='root'", "WS_PORT=''", `WS_GIT_NAME='O'\''Neil $(x)'`} {
		if !strings.Contains(got+"\n", want+"\n") {
			t.Errorf("no line %s in %q", want, got)
		}
	}
}
