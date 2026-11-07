package wizard

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
)

// On ws, the script goes on sh's stdin and each word goes as it is, with no ssh and no sudo.
func TestRemoteOnWsRunsTheScriptLocally(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{Shell: LocalShell{M: m}, Script: []byte("echo\n")}
	m.Results["sh -s -- status"] = effecttest.Result{Out: []byte("run abc 0 -\n")}
	if !r.Status(context.Background()) || r.Get("run") != "abc 0 -" {
		t.Fatalf("Status read %q; calls %q", r.Get("run"), m.Calls)
	}
	m.Results[wsPath+" server secrets set env A'B"] = effecttest.Result{Out: []byte("set\n")}
	if out, err := r.Secrets(context.Background(), []byte("dummy-value\n"), "set", "env", "A'B"); err != nil || string(out) != "set\n" {
		t.Fatalf("Secrets = %q, %v", out, err)
	}
	if want := []string{"echo\n", "dummy-value\n"}; !slices.Equal(m.Inputs, want) {
		t.Errorf("stdin = %q, want %q", m.Inputs, want)
	}
	if err := r.Put(context.Background(), "/var/lib/ws-wizard/x", []byte("data")); err != nil {
		t.Fatal(err)
	}
	if string(m.Files["/var/lib/ws-wizard/x"]) != "data" || m.Modes["/var/lib/ws-wizard/x"] != 0o600 {
		t.Errorf("Put wrote %q, mode %v", m.Files["/var/lib/ws-wizard/x"], m.Modes["/var/lib/ws-wizard/x"])
	}
}

func TestFollowOnWsHandsEventsOn(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{Shell: LocalShell{M: m}, Script: []byte("echo\n"), Events: &heard{}}
	m.Results["sh -s -- run --only=signoz/signoz up"] = effecttest.Result{Out: []byte(runLog)}
	if err := r.Follow(context.Background(), onlyArgs([]string{"signoz/signoz up"})...); err != nil {
		t.Fatal(err)
	}
	if got := r.Events.(*heard).steps(); !slices.Equal(got, []string{"ssh/keys"}) {
		t.Errorf("events = %q", got)
	}
	if want := []string{"ssh: changed", "last line"}; !slices.Equal(m.Passed, want) {
		t.Errorf("Passed = %q, want %q", m.Passed, want)
	}
}

// A Machine without effect.Lines still runs the run, its lines all passed on.
func TestFollowOnWsWithoutLines(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{Shell: LocalShell{M: struct{ effect.Machine }{m}}, Script: []byte("echo\n"), Events: &heard{}}
	m.Results["sh -s -- run"] = effecttest.Result{Out: []byte(runLog)}
	if err := r.Follow(context.Background()); err != nil || len(*r.Events.(*heard)) != 0 {
		t.Errorf("Follow = %v, events %v", err, r.Events)
	}
	if !slices.Equal(m.Calls, []string{"run sh -s -- run"}) {
		t.Errorf("calls = %q", m.Calls)
	}
}
