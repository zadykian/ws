package cli

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/ubuntu/values"
)

// tunnelOwn are the units that the host's own tunnel shares with a guest's.
var tunnelOwn = []string{"guard/tunnel", "state/tunnel-key", "service/tunnel", "state/tunnel-wg0"}

func hostTunnelUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	h, err := values.ParseHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	return loadSpecs(t, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: h, Side: "host", Self: selfPath})
}

func TestTheHostRunsItsOwnTunnelAndEachGuestsInOrder(t *testing.T) {
	names := kind.Names(hostTunnelUnits(t, tunnelMachine()))
	at := func(name string) int { return slices.Index(names, name) }
	order := append(slices.Clone(tunnelOwn), "state/tunnel-ufw", "guard/tunnel-guests", "state/tunnel-guest-keys", "state/tunnel-guest-links",
		"service/tunnel-networkd", "service/tunnel-forward", "state/tunnel-guest-forward", "state/tunnel-guest-ufw")
	for i := 1; i < len(order); i++ {
		if at(order[i-1]) < 0 || at(order[i]) < at(order[i-1]) {
			t.Fatalf("units %q, want %s before %s", names, order[i-1], order[i])
		}
	}
	if at("guard/tunnel-guests") < at("state/incus-guests") {
		t.Errorf("units %q, want the guests' tunnels after the guests", names)
	}
}

func TestAGuestHoldsNoHostTunnelUnit(t *testing.T) {
	lines := specLines(t, "guest", defaults(t))
	for _, unit := range []string{"state/tunnel-ufw", "guard/tunnel-guests", "state/tunnel-guest-keys", "state/tunnel-guest-links", "service/tunnel-networkd",
		"service/tunnel-forward", "state/tunnel-guest-forward", "state/tunnel-guest-ufw"} {
		if slices.Contains(lines, unit) {
			t.Errorf("the guest's lines hold %s", unit)
		}
	}
	if !slices.Contains(lines, "state/tunnel-wg0") {
		t.Error("the guest's lines lack its own wg0")
	}
}

func TestTheHostsOwnTunnelRunsTheHostsCommands(t *testing.T) {
	m := tunnelMachine()
	m.Results[selfPath+" ubuntu host tunnel check"] = effecttest.Result{Out: []byte("a line\n")}
	units := hostTunnelUnits(t, m)
	applyUnit(t, units, "guard/tunnel")
	if !slices.Contains(m.Calls, "root "+selfPath+" ubuntu host tunnel check") {
		t.Errorf("calls %q, want the host's check", m.Calls)
	}
}

func TestTheHostOpensItsOwnTunnelsPort(t *testing.T) {
	m := tunnelMachine()
	m.Files["/etc/ufw/user.rules"] = []byte("")
	got, err := planned(t, hostSpecs(t, m), "state/tunnel-ufw")
	if want := []string{"ufw allow in from any to any port 51820 proto udp"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("plan = %q, %v, want %q", got, err, want)
	}
}

func TestTheForwardServiceWritesItsUnitFile(t *testing.T) {
	m := tunnelMachine()
	applyUnit(t, hostTunnelUnits(t, m), "service/tunnel-forward")
	got := string(m.Files["/etc/systemd/system/tunnel-forward.service"])
	if !strings.Contains(got, "ExecStart=/usr/local/bin/ws ubuntu host tunnel guests forward\n") {
		t.Errorf("the unit file:\n%s", got)
	}
}

// networkd removes a foreign rule or route as it reconfigures a link. Without the drop-in, a guest's
// tunnel would lose its own.
func TestNetworkdKeepsTheRulesAndRoutesOfTheGuestsTunnels(t *testing.T) {
	m := tunnelMachine()
	applyUnit(t, hostTunnelUnits(t, m), "service/tunnel-networkd")
	got := string(m.Files["/etc/systemd/networkd.conf.d/60-ws-tunnel.conf"])
	for _, want := range []string{"[Network]\n", "ManageForeignRoutingPolicyRules=no\n", "ManageForeignRoutes=no\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("the drop-in lacks %q:\n%s", want, got)
		}
	}
}
