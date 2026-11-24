package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
)

// modeUnits are the units that a guest runs in the self-contained mode alone: SigNoz's, and the
// tunnel's own end. A guest under a host has both from the host.
var modeUnits = []string{
	"state/signoz-password", "compose/signoz", "state/signoz-sync",
	"guard/tunnel", "state/tunnel-key", "service/tunnel", "state/tunnel-wg0",
}

// A guest keeps the units of SigNoz and the tunnel whatever its values hold. It turns them off
// where the mode is. A unit that leaves the set would fail ws unit verify.
func TestAGuestRunsSigNozAndTheTunnelInTheModeAlone(t *testing.T) {
	for name, c := range map[string]struct {
		yml string
		off bool
	}{"off": {"---\n", true}, "on": {selfContained, false}} {
		m := tunnelMachine()
		m.Dirs["/opt/signoz"] = 0o755
		m.Results[selfPath+" ubuntu guest tunnel check"] = effecttest.Result{}
		units, out := guestUnits(t, m, c.yml)
		for _, unit := range modeUnits {
			if !slices.Contains(kind.Names(units), unit) {
				t.Fatalf("mode %s: no unit %s", name, unit)
			}
			out.Lines = nil
			p, err := units[slices.IndexFunc(units, func(u kind.Unit) bool { return u.Name == unit })].Step.Plan(t.Context())
			if err != nil {
				t.Fatalf("mode %s: %s: %v", name, unit, err)
			}
			if c.off && (len(p.Changes) != 0 || len(m.Calls) != 0) {
				t.Errorf("mode off: %s plans %d changes and calls %q, want none", unit, len(p.Changes), m.Calls)
			}
			if c.off != slices.ContainsFunc(out.Lines, func(l string) bool { return l == "say: "+unit+": off on this machine, as its when gives no set" }) {
				t.Errorf("mode %s: %s said %q", name, unit, out.Lines)
			}
		}
	}
}

// A host runs the same units whatever its values hold, as its file has no mode.
func TestAHostRunsSigNozAndTheTunnelAlways(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[selfPath+" ubuntu host tunnel check"] = effecttest.Result{}
	out := &effecttest.Out{}
	steps := hostSteps(t, m, out)
	for _, unit := range modeUnits {
		if _, ok := steps[unit]; !ok {
			t.Errorf("the host has no unit %s", unit)
		}
	}
	for _, unit := range modeUnits {
		if _, err := planned(t, steps, unit); err != nil && !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("%s: %v", unit, err)
		}
	}
	for _, l := range out.Lines {
		if strings.Contains(l, "off on this machine") && slices.ContainsFunc(modeUnits, func(u string) bool { return strings.Contains(l, u) }) {
			t.Errorf("the host turned a unit off: %q", l)
		}
	}
}
