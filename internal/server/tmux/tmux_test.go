package tmux

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

const list = "Name  Version  Rev  Tracking  Publisher  Notes\ntmux  3.5a  1  latest/stable  x  classic\n"

func snapMachine(booted bool, snaps string) (*effecttest.Machine, *effecttest.Out) {
	m := effecttest.NewMachine()
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	m.Files["/usr/bin/snap"] = []byte("x")
	m.Results["snap list"] = effecttest.Result{Out: []byte(snaps)}
	m.Results["snap remove tmux"] = effecttest.Result{}
	return m, &effecttest.Out{}
}

func TestSnapIsRemovedOnce(t *testing.T) {
	m, out := snapMachine(true, list)
	changes, err := Snap{M: m, Out: out}.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("got %d changes, %v", len(changes), err)
	}
	if err := changes[0].(interface{ Apply(context.Context) error }).Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := m.Calls[len(m.Calls)-1]; got != "root snap remove tmux" {
		t.Errorf("ran %q", got)
	}
	m, out = snapMachine(true, "Name  Version\ncore  1\n")
	if changes, err := (Snap{M: m, Out: out}).Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("without the snap: %v, %v", changes, err)
	}
}

func TestSnapSkipsWithoutSystemdOrSnapd(t *testing.T) {
	m, out := snapMachine(false, list)
	if changes, err := (Snap{M: m, Out: out}).Plan(context.Background()); err != nil || len(changes) != 0 || len(out.Lines) != 1 {
		t.Errorf("no systemd: %v, %v, said %q", changes, err, out.Lines)
	}
	m, out = snapMachine(true, list)
	delete(m.Files, "/usr/bin/snap")
	if changes, err := (Snap{M: m, Out: out}).Plan(context.Background()); err != nil || len(changes) != 0 || slices.Contains(m.Calls, "run snap list") {
		t.Errorf("no snapd: %v, %v, calls %q", changes, err, m.Calls)
	}
}

func TestStepsAreTheRole(t *testing.T) {
	steps := Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Out{})
	if len(steps) != 1 || steps[0].Name() != "Tmux" {
		t.Errorf("got %d steps, want Tmux alone", len(steps))
	}
}
