package security

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestOutboundWritesTheFillAndTheTableThenLoadsThem(t *testing.T) {
	v := values.Default()
	v.Home = "/root"
	m, u, ss := fakeSteps(v)
	run(t, ss)
	for _, p := range []string{scriptFile, tableFile, "/etc/systemd/system/" + outboundUnit,
		"/etc/systemd/system/" + pathUnit, "/etc/systemd/system/" + timerUnit, "/etc/systemd/system/" + fillUnit} {
		if _, ok := m.Files[p]; !ok {
			t.Errorf("%s was not written", p)
		}
	}
	if got := string(m.Files["/etc/systemd/system/"+outboundUnit]); !strings.Contains(got, "ExecStart="+scriptFile+" "+tableFile+"\n") {
		t.Errorf("the unit does not load the table through the script:\n%s", got)
	}
	for _, want := range []string{"enable " + pathUnit, "start " + pathUnit, "enable " + outboundUnit, "start " + outboundUnit} {
		if !slices.Contains(u.Calls, want) {
			t.Errorf("calls = %q, want %q", u.Calls, want)
		}
	}
	if !slices.Contains(u.Calls, "daemon-reload") {
		t.Errorf("calls = %q, want a daemon-reload", u.Calls)
	}
}

func TestOutboundOpensDockersBridgesToRootAlone(t *testing.T) {
	v := values.Default()
	v.Home = "/root"
	m, _, ss := fakeSteps(v)
	run(t, ss)
	table := string(m.Files[tableFile])
	for _, want := range []string{"meta skuid 0 oifname \"docker0\" accept", "meta skuid 0 oifname \"br-*\" accept"} {
		if !strings.Contains(table, want) {
			t.Errorf("the table lacks %q", want)
		}
	}
	if strings.Contains(table, "{ \"lo\", \"ikeward0\", \"docker0\"") {
		t.Error("the table opens docker0 to every user")
	}
}

func TestOutboundRunsNftCBeforeTheTableIsInPlace(t *testing.T) {
	v := values.Default()
	v.Home = "/root"
	m, _, ss := fakeSteps(v)
	run(t, ss)
	if !slices.Contains(m.Calls, "root /usr/sbin/nft -c -f "+tableFile+".ws-check") {
		t.Errorf("calls = %q, want nft -c on the pending file", m.Calls)
	}
}

func TestOutboundOffTheOptionLeavesTheStepOut(t *testing.T) {
	v := values.Default()
	v.Home, v.Security.Outbound = "/root", false
	_, _, ss := fakeSteps(v)
	for _, s := range ss {
		if s.Name() == outboundStep {
			t.Errorf("step %q is there with the table off", s.Name())
		}
	}
}

func TestOutboundReloadsWhereTheKernelsTableIsNotTheFiles(t *testing.T) {
	v := values.Default()
	v.Home = "/root"
	m, _, ss := fakeSteps(v)
	run(t, ss)
	m.Results["nft -s list table inet ws"] = effecttest.Result{Out: []byte("other\n")}
	for _, s := range ss {
		if s.Name() != outboundStep {
			continue
		}
		p, err := s.Plan(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, c := range p.Changes {
			got = append(got, c.String())
		}
		for _, want := range []string{"reload " + outboundUnit, "restart " + fillUnit} {
			if !slices.Contains(got, want) {
				t.Errorf("changes = %q, want %q", got, want)
			}
		}
	}
}

// offFixture is a machine that an earlier run left the table and its units on.
func offFixture() (*effecttest.Machine, *effecttest.Units, []step.Step) {
	v := values.Default()
	v.Home, v.Security.Outbound = "/root", false
	m, u, ss := fakeSteps(v)
	for _, n := range outboundUnits {
		m.Files["/etc/systemd/system/"+n] = []byte("[Unit]\n")
	}
	m.Dirs["/etc/systemd/system"] = 0o755
	m.Dirs["/etc/systemd/system/sysinit.target.wants"] = 0o755
	m.Links["/etc/systemd/system/sysinit.target.wants/"+pathUnit] = "/etc/systemd/system/" + pathUnit
	m.Files[tableFile] = []byte("table\n")
	m.Results["nft list table inet ws"] = effecttest.Result{Out: []byte("table\n")}
	m.Dirs[tableDir] = 0o755
	u.States = map[string]effect.UnitState{
		outboundUnit: {Active: true}, pathUnit: {Active: true, Enabled: true}, fillUnit: {Failed: true},
	}
	return m, u, ss
}

// offChanges plans the step that takes the table off.
func offChanges(t *testing.T, ss []step.Step) []string {
	for _, s := range ss {
		if s.Name() != offStep {
			continue
		}
		p, err := s.Plan(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, c := range p.Changes {
			got = append(got, c.String())
		}
		return got
	}
	t.Fatal("no step takes the table off")
	return nil
}

func TestOutboundOffStopsTheUnitsThenRemovesThemAndTheTable(t *testing.T) {
	_, _, ss := offFixture()
	got := offChanges(t, ss)
	for _, want := range []string{"stop " + pathUnit, "stop " + outboundUnit, "reset failed state of " + fillUnit,
		"remove /etc/systemd/system/" + outboundUnit, "remove " + tableDir,
		"remove /etc/systemd/system/sysinit.target.wants/" + pathUnit,
		"reload systemd's unit files", "remove the outbound table"} {
		if !slices.Contains(got, want) {
			t.Errorf("changes = %q, want %q", got, want)
		}
	}
}

func TestOutboundOffPlansNothingOnceTheUnitsAreGone(t *testing.T) {
	v := values.Default()
	v.Home, v.Security.Outbound = "/root", false
	m, _, ss := fakeSteps(v)
	m.Dirs["/etc/systemd/system"] = 0o755
	m.Results["nft list table inet ws"] = effecttest.Result{Err: errors.New("no table")}
	if got := offChanges(t, ss); len(got) > 0 {
		t.Errorf("changes = %q, want none", got)
	}
}
