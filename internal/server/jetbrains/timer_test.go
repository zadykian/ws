package jetbrains

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestTimerWritesTheUnitsAndEnablesIt(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files["/home/u/.local/bin/ws"] = []byte("ws binary")
	u := &effecttest.Units{}
	parts, err := timer(m, u, &effecttest.Out{}, "/home/u/.local/bin/ws")(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var changes []host.Change
	for _, p := range parts {
		cs, err := p.Plan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, cs...)
	}
	if len(changes) < 6 {
		t.Fatalf("%d changes: %v", len(changes), changes)
	}
	for _, c := range changes {
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Join(u.Calls, ","); !strings.Contains(got, "daemon-reload") || !strings.Contains(got, "enable "+timerName) ||
		!strings.Contains(got, "start "+timerName) {
		t.Errorf("units: %s", got)
	}
	svc := string(m.Files[serviceFile])
	if !strings.Contains(svc, "ExecStart=/usr/local/lib/ws/jetbrains server apply --only jetbrains\n") ||
		!strings.Contains(svc, "WS_JETBRAINS_RUN=timer") {
		t.Errorf("service:\n%s", svc)
	}
	if string(m.Files[binaryFile]) != "ws binary" {
		t.Error("root's copy of ws is not written")
	}
	if !strings.Contains(string(m.Files[timerFile]), "OnCalendar=weekly") {
		t.Error("no weekly calendar")
	}
}
