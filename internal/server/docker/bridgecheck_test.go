package docker

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func held(prefixes ...string) func() ([]netip.Prefix, error) {
	return func() ([]netip.Prefix, error) {
		var out []netip.Prefix
		for _, p := range prefixes {
			out = append(out, netip.MustParsePrefix(p))
		}
		return out, nil
	}
}

func TestBridgeCheckPlansAMoveAsAChangeThatFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	for _, c := range []struct{ held, want string }{
		{"", "no address"}, {"172.17.0.1/24", "172.17.0.1/24"}, {"10.1.0.1/16", "10.1.0.1/16"},
	} {
		var addrs []string
		if c.held != "" {
			addrs = []string{"192.168.0.1/24", c.held}
		}
		check := bridgeCheck{m: m, out: &effecttest.Out{}, held: held(addrs...), want: "172.17.0.1/16"}
		changes, err := check.Plan(t.Context())
		if err != nil || len(changes) != 1 || !strings.Contains(changes[0].String(), "holds ") {
			t.Fatalf("held %q: plan = %v, %v; want one change", c.held, changes, err)
		}
		if err := changes[0].Apply(t.Context()); err == nil || !strings.Contains(err.Error(), "docker0 holds") {
			t.Errorf("Apply = %v, want an error that names what docker0 holds", err)
		}
	}
}

func TestBridgeCheckPassesOnTheWantedAddressAndWithoutSystemd(t *testing.T) {
	m := effecttest.NewMachine()
	out := &effecttest.Out{}
	check := bridgeCheck{m: m, out: out, held: held("172.17.0.1/16"), want: "172.17.0.1/16"}
	if changes, err := check.Plan(t.Context()); err != nil || changes != nil || len(out.Lines) != 1 {
		t.Errorf("without systemd: %v, %v, said %q; want a skip", changes, err, out.Lines)
	}
	m.Dirs["/run/systemd/system"] = 0o755
	if changes, err := check.Plan(t.Context()); err != nil || changes != nil {
		t.Errorf("with the address: %v, %v; want none", changes, err)
	}
}
