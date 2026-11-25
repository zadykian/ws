package values

import (
	"strings"
	"testing"
)

func TestAssignments(t *testing.T) {
	got := Conf{Target: "lan", Slot: "0", CableService: "O'Neil $(x)"}.Assignments()
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != len(keys) {
		t.Fatalf("%d lines, want %d: %q", len(lines), len(keys), got)
	}
	for _, want := range []string{"WS_TARGET='lan'", "WS_SLOT='0'", "WS_TUNNEL_PORT=''", `WS_CABLE_SERVICE='O'\''Neil $(x)'`} {
		if !strings.Contains(got+"\n", want+"\n") {
			t.Errorf("no line %s in %q", want, got)
		}
	}
}
