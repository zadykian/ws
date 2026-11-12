package ikeward

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
)

const (
	addPeer  = "ufw route allow in on wg0 out on ikeward0 from 10.99.0.2 to any comment ikeward"
	dropPeer = "ufw --force route delete allow in on wg0 out on ikeward0 from 10.99.0.9 to any comment ikeward"
	userRule = "### tuple ### route:allow any any 0.0.0.0/0 any 10.99.0.9 in_wg0!out_ikeward0 comment=696b6577617264\n"
)

func ufwMachine(forwarding string) *effecttest.Machine {
	m := forwardMachine(forwarding)
	m.Files["/etc/ufw/user.rules"] = []byte(userRule)
	m.Results[addPeer], m.Results[dropPeer] = effecttest.Result{}, effecttest.Result{}
	return m
}

func TestForwardSwapsThePeersRules(t *testing.T) {
	m := ufwMachine("1\n")
	m.Files[sysctlPath] = sysctlFile
	apply(t, forward(m, &effecttest.Out{}, []string{"10.99.0.2"}))
	for _, want := range []string{"root " + addPeer, "root " + dropPeer} {
		if !slices.Contains(m.Calls, want) {
			t.Errorf("calls = %q, lack %q", m.Calls, want)
		}
	}
}

func TestForwardWithNoPeerRemovesTheFileAndTheRules(t *testing.T) {
	m := ufwMachine("1\n")
	m.Files[sysctlPath] = sysctlFile
	apply(t, forward(m, &effecttest.Out{}, nil))
	for _, want := range []string{"root rm -rf -- " + sysctlPath, "root " + dropPeer} {
		if !slices.Contains(m.Calls, want) {
			t.Errorf("calls = %q, lack %q", m.Calls, want)
		}
	}
}

func TestForwardPlansNothingOnceSettled(t *testing.T) {
	m := ufwMachine("1\n")
	m.Files[sysctlPath] = sysctlFile
	g := host.Group{Title: "forward", Parts: forward(m, &effecttest.Out{}, []string{"10.99.0.9"})}
	if p, err := g.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v; want nothing", p.Changes, err)
	}
}

func TestOwnsOnlyTheRoutesFromWgToIkeward(t *testing.T) {
	if !owns(routes([]string{"10.99.0.2"})[0]) {
		t.Error("a peer's route is not owned")
	}
	other := routes([]string{"10.99.0.2"})[0]
	other.Interface = "eth0!out_ikeward0"
	if owns(other) {
		t.Error("a route from another interface is owned")
	}
}
