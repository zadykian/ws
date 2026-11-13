package sessiontest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
)

// The text that a test states reads back as its value.
func TestTextReadsBack(t *testing.T) {
	full := session.Status{Run: session.Stamp{Version: "v1", Exit: "0", Off: true, Secrets: "aa", Vars: "bb"},
		Active: "u", Version: "v1", User: "alice", MacKey: "mk", WgKey: "wk", Signing: "none", Hostname: "ws",
		Claude: "ok", FansRefused: true, Cable: "p", Peered: true, OutboundOff: true, Secrets: "s", Vars: "v"}
	for _, want := range []session.Status{{}, full} {
		m := effecttest.NewMachine()
		got, ok := On(m, want).Status(context.Background())
		if !ok || got != want {
			t.Errorf("Status = %+v, %v; want %+v", got, ok, want)
		}
	}
}

func script(op ...string) []string { return append([]string{"sh", "-s", "--"}, op...) }

func TestWsKeysAScriptByItsOp(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[Op("link", "eth1")] = effecttest.Result{Out: []byte("ok")}
	m.Results["uname -r"] = effecttest.Result{Out: []byte("7.0")}
	var f remote.Shell = Ws{M: m}
	ctx := context.Background()
	if out, err := f.Run(ctx, []byte("echo\n"), script("link", "eth1")...); err != nil || string(out) != "ok" {
		t.Errorf("Run of the op = %q, %v", out, err)
	}
	if out, err := f.Run(ctx, nil, "uname", "-r"); err != nil || string(out) != "7.0" {
		t.Errorf("Run of a plain command = %q, %v", out, err)
	}
	if got := m.Inputs[0]; got != "echo\n" {
		t.Errorf("the script on stdin = %q", got)
	}
}

func TestWsPasses(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[Op("install", "v1")] = effecttest.Result{Err: errors.New("denied")}
	m.Results[Op("run")] = effecttest.Result{Out: []byte("event\nother\n")}
	var f remote.Shell = Ws{M: m}
	ctx := context.Background()
	if err := f.Pass(ctx, nil, script("install", "v1")...); err == nil {
		t.Error("Pass of the op: err = nil")
	}
	if err := f.Lines(ctx, nil, func(line string) bool { return line == "event" }, script("run")...); err != nil {
		t.Fatal(err)
	}
	if want := []string{"other"}; !slices.Equal(m.Passed, want) {
		t.Errorf("passed = %q; want %q", m.Passed, want)
	}
}

func TestWsKeepsAWordWhole(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[Op("run", "--only", "network")] = effecttest.Result{Out: []byte("two")}
	m.Results[Op("run", "--only network")] = effecttest.Result{Out: []byte("one")}
	var f remote.Shell = Ws{M: m}
	for want, op := range map[string][]string{"two": {"run", "--only", "network"}, "one": {"run", "--only network"}} {
		if out, err := f.Run(context.Background(), nil, script(op...)...); err != nil || string(out) != want {
			t.Errorf("Run of %q = %q, %v; want %q", op, out, err, want)
		}
	}
}
