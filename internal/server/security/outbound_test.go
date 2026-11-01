package security

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
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
