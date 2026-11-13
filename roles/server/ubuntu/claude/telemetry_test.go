package claude

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
)

var rootUser = host.User{Name: "root", Home: "/root"}

// settingsReadScript is the script that asuser.Read runs.
const settingsReadScript = `if [ -e "$1" ]; then printf y; cat "$1"; else printf n; fi`

// serveSettings fakes the command that asuser.Read runs for the user's file. A nil body is no file.
func serveSettings(m *effecttest.Machine, u host.User, file string, body []byte) {
	out := []byte("n")
	if body != nil {
		out = append([]byte("y"), body...)
	}
	line := "env HOME=" + u.Home + " sh -c " + settingsReadScript + " sh " + file
	if u.Name != "root" {
		line = "runuser -u " + u.Name + " -- env -C " + u.Home + " " + strings.TrimPrefix(line, "env ")
	}
	m.Results[line] = effecttest.Result{Out: out}
}

func TestTelemetryPlan(t *testing.T) {
	m := effecttest.NewMachine()
	tel := Telemetry{M: m, User: rootUser, Env: map[string]string{"A": "1"}}
	serveSettings(m, rootUser, tel.path(), nil)
	m.Results[`env HOME=/root sh -c umask 022 && mkdir -p "${1%/*}" && cat >"$1" sh `+tel.path()] = effecttest.Result{}
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
	serveSettings(m, rootUser, tel.path(), []byte(m.Inputs[0]))
	if changes, err = tel.Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Fatalf("second plan: %v %v", changes, err)
	}
}

func TestTelemetryRefusesOtherShape(t *testing.T) {
	m := effecttest.NewMachine()
	serveSettings(m, rootUser, rootUser.Home+"/.claude/settings.json", []byte(`[]`))
	tel := Telemetry{M: m, User: rootUser, Env: map[string]string{"A": "1"}}
	if _, err := tel.Plan(context.Background()); err == nil {
		t.Fatal("no error")
	}
}
