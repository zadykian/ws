package route_test

import (
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/route"
)

const ipRoutes = "ip -j -4 route show table all"

// clearOf checks a pool 10.1.0.0/16 and docker0's network 172.17.0.1/16. The machine's routes
// are the JSON routes.
func clearOf(routes string) (route.Clear, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Results[ipRoutes] = effecttest.Result{Out: []byte(routes)}
	return route.Clear{M: m, Pool: netip.MustParsePrefix("10.1.0.0/16"), Net: netip.MustParsePrefix("172.17.0.1/16"),
		Dev: "docker0", Hint: "Move the pool."}, m
}

func TestClearRefusesAnOverlap(t *testing.T) {
	for routes, want := range map[string]string{
		`[{"dst":"10.0.0.0/8","dev":"eth0"}]`:               "10.0.0.0/8 dev eth0 table main",
		`[{"dst":"10.1.5.0/24","dev":"wg0","table":"7"}]`:   "10.1.5.0/24 dev wg0 table 7",
		`[{"dst":"172.17.3.4","dev":"tun0"}]`:               "172.17.3.4/32 dev tun0 table main",
		`[{"dst":"172.17.0.0/16","dev":"br-1","scope":""}]`: "172.17.0.0/16 dev br-1 table main",
	} {
		c, _ := clearOf(routes)
		_, err := c.Plan(t.Context())
		if err == nil || !strings.Contains(err.Error(), want) || !strings.HasSuffix(err.Error(), ". Move the pool.") {
			t.Errorf("routes %s: Plan = %v, want an error that names %s and ends with the hint", routes, err, want)
		}
	}
}

func TestClearPassesOwnDeviceAndNoOverlap(t *testing.T) {
	c, _ := clearOf(`[{"dst":"default","dev":"eth0"},` +
		`{"dst":"172.17.0.0/16","dev":"docker0","scope":"link"},` +
		`{"dst":"10.1.2.0/24","dev":"eth1","scope":"link"},` +
		`{"dst":"0.0.0.0/1","dev":"wg0","table":"51820"},` +
		`{"dst":"10.1.0.0/16","type":"local","dev":"lo","table":"local"},` +
		`{"dst":"192.168.1.0/24","dev":"eth0","scope":"link"}]`)
	if changes, err := c.Plan(t.Context()); err != nil || changes != nil {
		t.Errorf("Plan = %v, %v; want none: docker0's own route, an on-link route of the main table, "+
			"a catch-all, a local route and a clear network", changes, err)
	}
}

func TestClearFailsWithoutTheRoutes(t *testing.T) {
	c, m := clearOf("")
	m.Results[ipRoutes] = effecttest.Result{Err: errors.New("ip: not found")}
	if _, err := c.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "read the routes") {
		t.Errorf("Plan = %v, want the read of the routes to fail", err)
	}
}

// addr is the check that docker0 holds 172.17.0.1/16, where docker0 holds the addresses held.
func addr(held ...string) route.Addr {
	return route.Addr{Iface: "docker0", Want: netip.MustParsePrefix("172.17.0.1/16"), Hint: "Reboot.",
		Held: func(iface string) ([]netip.Prefix, error) {
			if iface != "docker0" {
				return nil, errors.New("another interface")
			}
			var got []netip.Prefix
			for _, h := range held {
				got = append(got, netip.MustParsePrefix(h))
			}
			return got, nil
		}}
}

func TestAddrRefusesAMissingAddress(t *testing.T) {
	for _, c := range []struct {
		held []string
		want string
	}{
		{nil, "docker0 holds no address, not 172.17.0.1/16"},
		{[]string{"172.18.0.1/16", "172.17.0.1/24"}, "docker0 holds 172.18.0.1/16, 172.17.0.1/24, not 172.17.0.1/16"},
	} {
		a := addr(c.held...)
		changes, err := a.Plan(t.Context())
		if err != nil || len(changes) != 1 || changes[0].String() != c.want {
			t.Fatalf("held %q: Plan = %v, %v; want %q", c.held, changes, err, c.want)
		}
		if err := changes[0].Apply(t.Context()); err == nil || !strings.HasSuffix(err.Error(), ": Reboot.") {
			t.Errorf("held %q: Apply = %v, want a failure that ends with the hint", c.held, err)
		}
	}
}

func TestAddrPassesTheWantedAddress(t *testing.T) {
	a := addr("10.0.0.1/8", "172.17.0.1/16")
	if changes, err := a.Plan(t.Context()); err != nil || changes != nil {
		t.Errorf("Plan = %v, %v; want none", changes, err)
	}
}
