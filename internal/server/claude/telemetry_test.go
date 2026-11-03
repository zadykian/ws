package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestTelemetryPlan(t *testing.T) {
	m := effecttest.NewMachine()
	tel := Telemetry{M: m, User: root, Env: map[string]string{"A": "1"}}
	m.Results[`env HOME=/root sh -c umask 022 && cat >"$1" sh `+tel.path()] = effecttest.Result{}
	changes, err := tel.Plan(context.Background())
	if err != nil || len(changes) != 1 || !strings.Contains(changes[0].String(), "create") {
		t.Fatalf("new file: %v %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(m.Inputs) != 1 || m.Files[tel.path()] != nil {
		t.Fatalf("not written as the user through a command: %q", m.Inputs)
	}
	m.Files[tel.path()] = []byte(m.Inputs[0])
	if changes, err = tel.Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Fatalf("second plan: %v %v", changes, err)
	}
}

func TestTelemetryRefusesOtherShape(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[root.Home+"/.claude/settings.json"] = []byte(`[]`)
	tel := Telemetry{M: m, User: root, Env: map[string]string{"A": "1"}}
	if _, err := tel.Plan(context.Background()); err == nil {
		t.Fatal("no error")
	}
}
