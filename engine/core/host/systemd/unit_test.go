package systemd_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/systemd"
	"github.com/zadykian/ws/engine/core/step"
)

func unitFixture(booted bool) (*effecttest.Machine, *effecttest.Units, *effecttest.Out) {
	m := effecttest.NewMachine()
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	return m, &effecttest.Units{}, &effecttest.Out{}
}

func TestUnitEnablesAndStarts(t *testing.T) {
	m, u, out := unitFixture(true)
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: systemd.Running}
	if got := applyAll(t, unit); !slices.Equal(got, []string{"enable a.service", "start a.service"}) {
		t.Errorf("first plan = %q, want enable and start", got)
	}
	if got := applyAll(t, unit); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestUnitSkipsWithoutSystemd(t *testing.T) {
	m, u, out := unitFixture(false)
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: systemd.Running}
	if got := applyAll(t, unit); got != nil || len(u.Calls) != 0 || len(out.Lines) != 1 {
		t.Errorf("plan = %q, calls %v, said %q; want a skip said once", got, u.Calls, out.Lines)
	}
}

func TestUnitRestartsAfterAWatchedChange(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}}
	conf := host.File{M: m, Path: "/srv/a.conf", Content: []byte("x"), Mode: 0o644}
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: systemd.Running, Watch: []host.Resource{conf}}
	g := host.Group{Title: "A", Parts: []host.Resource{conf, unit}}
	if _, err := step.Run(t.Context(), []step.Step{g}, out, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(u.Calls, []string{"restart a.service"}) || string(m.Files["/srv/a.conf"]) != "x" {
		t.Errorf("calls = %q, want one restart after the write", u.Calls)
	}
}

func TestUnitResetsAFailureThenStarts(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Failed: true}, "b.service": {Active: true}}
	a := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Run: systemd.Running}
	b := systemd.Unit{M: m, U: u, Out: out, Name: "b.service", Run: systemd.Stopped}
	applyAll(t, a)
	applyAll(t, b)
	if want := []string{"reset-failed a.service", "start a.service", "stop b.service"}; !slices.Equal(u.Calls, want) {
		t.Errorf("calls = %q, want %q", u.Calls, want)
	}
}

func TestDaemonReloadFollowsAUnitFile(t *testing.T) {
	m, u, out := unitFixture(true)
	unitFile := host.File{M: m, Path: "/srv/systemd/system/a.service", Content: []byte("[Unit]\n"), Mode: 0o644}
	g := host.Group{Title: "A", Parts: []host.Resource{unitFile, systemd.DaemonReload{M: m, U: u, Watch: []host.Resource{unitFile}}}}
	for range 2 {
		if _, err := step.Run(t.Context(), []step.Step{g}, out, false); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(u.Calls, []string{"daemon-reload"}) {
		t.Errorf("calls = %q, want one daemon-reload", u.Calls)
	}
}

func TestUnitReloadsWhereAFileWasWrittenAfterACutShortRun(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true, NeedsReload: true}}
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: systemd.Running}
	if got := applyAll(t, unit); !slices.Equal(got, []string{"reload systemd's unit files"}) {
		t.Errorf("first plan = %q, want a reload", got)
	}
	if got := applyAll(t, unit); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestUnitRestartIsForced(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Active: true}}
	conf := host.File{M: m, Path: "/srv/a.conf", Content: []byte("x"), Mode: 0o644}
	cs, err := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Run: systemd.Running, Watch: []host.Resource{conf}}.Plan(t.Context())
	f, ok := cs[len(cs)-1].(host.Forced)
	if err != nil || len(cs) != 2 || !ok || !f.Forced() {
		t.Errorf("plan = %v, %v; want a forced restart", cs, err)
	}
}

func TestUnitReloadsAfterAWatchedChange(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}}
	conf := host.File{M: m, Path: "/srv/a.conf", Content: []byte("x"), Mode: 0o644}
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: systemd.Running, Reload: true,
		Watch: []host.Resource{conf}}
	g := host.Group{Title: "A", Parts: []host.Resource{conf, unit}}
	if _, err := step.Run(t.Context(), []step.Step{g}, out, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(u.Calls, []string{"reload a.service"}) {
		t.Errorf("calls = %q, want one reload", u.Calls)
	}
}

func TestUnitDisables(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}}
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Disabled: true, Run: systemd.Stopped}
	if got := applyAll(t, unit); !slices.Equal(got, []string{"disable a.service", "stop a.service"}) {
		t.Errorf("first plan = %q, want disable and stop", got)
	}
	if got := applyAll(t, unit); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestUnitRestartSurvivesACutShortRun(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Active: true}}
	conf := host.File{M: m, Path: "/srv/a.conf", Content: []byte("x"), Mode: 0o644}
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Run: systemd.Running, Watch: []host.Resource{conf}}
	g := host.Group{Title: "A", Parts: []host.Resource{conf, unit}}
	p, err := g.Plan(t.Context())
	if err != nil || len(p.Changes) != 3 {
		t.Fatalf("plan = %v, %v; want the mark, the write and the restart", p.Changes, err)
	}
	// The run ends after the write: the restart never runs.
	for _, c := range p.Changes[:2] {
		if err := c.(host.Change).Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := step.Run(t.Context(), []step.Step{g}, out, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(u.Calls, []string{"restart a.service"}) {
		t.Errorf("calls = %q, want the restart that the cut run owed", u.Calls)
	}
}

func TestUnitFailedRestartStaysOwed(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Active: true}}
	conf := host.File{M: m, Path: "/srv/a.conf", Content: []byte("x"), Mode: 0o644}
	unit := systemd.Unit{M: m, U: failingRestart{u}, Out: out, Name: "a.service", Run: systemd.Running, Watch: []host.Resource{conf}}
	g := host.Group{Title: "A", Parts: []host.Resource{conf, unit}}
	if _, err := step.Run(t.Context(), []step.Step{g}, out, false); err == nil {
		t.Fatal("run succeeded, want the restart's failure")
	}
	if cs, err := unit.Plan(t.Context()); err != nil || len(cs) != 1 {
		t.Errorf("plan after a failed restart = %v, %v; want the restart again", cs, err)
	}
}

type failingRestart struct{ *effecttest.Units }

func (failingRestart) Restart(context.Context, string) error { return errors.New("boom") }

func TestUnitTriggeredRestartsOnAWatchedChangeAlone(t *testing.T) {
	m, u, out := unitFixture(true)
	conf := host.File{M: m, Path: "/srv/a.conf", Content: []byte("x"), Mode: 0o644}
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Run: systemd.Triggered}
	if got := applyAll(t, unit); got != nil {
		t.Fatalf("plan = %q, want none: nothing is watched", got)
	}
	unit.Watch = []host.Resource{conf}
	g := host.Group{Title: "A", Parts: []host.Resource{conf, unit}}
	if _, err := step.Run(t.Context(), []step.Step{g}, out, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(u.Calls, []string{"restart a.service"}) {
		t.Errorf("calls = %q, want one restart", u.Calls)
	}
}

func TestUnitStoppedResetsAFailure(t *testing.T) {
	m, u, out := unitFixture(true)
	u.States = map[string]effect.UnitState{"a.service": {Failed: true}}
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Run: systemd.Stopped}
	if got := applyAll(t, unit); !slices.Equal(got, []string{"reset failed state of a.service"}) {
		t.Errorf("plan = %q, want a reset", got)
	}
}

func TestUnitReport(t *testing.T) {
	m, u, out := unitFixture(false)
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}, "b.service": {Failed: true}}
	unit := systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: systemd.Running}
	if got := reportOf(t, unit); got.Found != "systemd is not running" || len(out.Lines) != 0 {
		t.Errorf("item = %+v, said %q; want the state, left for Plan to say", got, out.Lines)
	}
	m.Dirs["/run/systemd/system"] = 0o755
	tests := []struct {
		unit        systemd.Unit
		want, found string
	}{
		{unit, "enabled, running", "enabled, running"},
		{systemd.Unit{M: m, U: u, Name: "b.service", Disabled: true, Run: systemd.Stopped}, "disabled, stopped", "disabled, failed"},
		{systemd.Unit{M: m, U: u, Name: "b.service", Run: systemd.Triggered}, "started by its trigger", "disabled, failed"},
		{systemd.Unit{M: m, U: u, Name: "b.service"}, "as it is", "disabled, failed"},
	}
	for _, tt := range tests {
		if got := reportOf(t, tt.unit); got.Want != tt.want || got.Found != tt.found {
			t.Errorf("item = %+v, want %q, found %q", got, tt.want, tt.found)
		}
	}
}
