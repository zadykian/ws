package routing

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

func TestRunRoutes(t *testing.T) {
	a, b, c := netip.MustParsePrefix("10.1.0.0/16"), netip.MustParsePrefix("10.2.0.0/16"), netip.MustParsePrefix("10.3.0.0/16")
	m := effecttest.NewMachine()
	for _, n := range []netip.Prefix{a, b} {
		m.Results["route -n get -inet "+n.String()] = effecttest.Result{Out: []byte(getOn(n, "utun4"))}
	}
	m.Results["route -q -n delete -inet 10.1.0.0/16"] = effecttest.Result{Err: errors.New("not in table")}
	m.Results["route -q -n delete -inet 10.2.0.0/16"] = effecttest.Result{}
	m.Results["route -q -n add -inet 10.3.0.0/16 -interface utun4"] = effecttest.Result{Err: errors.New("exists")}
	m.Results["route -q -n add -inet 10.4.0.0/16 -interface utun4"] = effecttest.Result{}
	d := netip.MustParsePrefix("10.4.0.0/16")
	failed, logs := RunRoutes(context.Background(), m, tunnel.RoutePlan{Delete: []netip.Prefix{a, b}, Add: []netip.Prefix{c, d}}, "utun4")
	if !slices.Equal(failed, []netip.Prefix{c}) {
		t.Errorf("failed = %v, want [%v]", failed, c)
	}
	if !slices.Equal(logs, []string{"cannot remove the route to 10.1.0.0/16"}) {
		t.Errorf("logs = %q", logs)
	}
	want := []string{
		"run route -n get -inet 10.1.0.0/16", "run route -q -n delete -inet 10.1.0.0/16",
		"run route -n get -inet 10.2.0.0/16", "run route -q -n delete -inet 10.2.0.0/16",
		"run route -q -n add -inet 10.3.0.0/16 -interface utun4", "run route -q -n add -inet 10.4.0.0/16 -interface utun4",
	}
	if !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

// getOn is route -n get's answer for the route to n through the interface.
func getOn(n netip.Prefix, iface string) string {
	a := n.Addr().String()
	return "   route to: " + a + "\ndestination: " + a + "\n       mask: " + net.IP(net.CIDRMask(n.Bits(), 32)).String() +
		"\n  interface: " + iface + "\n      flags: <UP,DONE,STATIC,PRCLONING>\n"
}

// getDefault is route -n get's answer where the default route, through en0, is the best match.
const getDefault = "   route to: 10.4.0.0\ndestination: default\n       mask: default\n    gateway: 192.168.1.1\n" +
	"  interface: en0\n      flags: <UP,GATEWAY,DONE,STATIC,PRCLONING>\n"

func TestRunRoutesDeletesOnTheDaemonsOwnUtunAlone(t *testing.T) {
	own, other, none, deflt := netip.MustParsePrefix("10.1.0.0/16"), netip.MustParsePrefix("10.2.0.0/16"),
		netip.MustParsePrefix("10.3.0.0/16"), netip.MustParsePrefix("10.4.0.0/16")
	m := effecttest.NewMachine()
	m.Results["route -n get -inet 10.1.0.0/16"] = effecttest.Result{Out: []byte(getOn(own, "utun4"))}
	m.Results["route -n get -inet 10.2.0.0/16"] = effecttest.Result{Out: []byte(getOn(other, "utun5"))}
	m.Results["route -q -n delete -inet 10.1.0.0/16"] = effecttest.Result{}
	m.Results["route -n get -inet 10.3.0.0/16"] = effecttest.Result{Err: errors.New("not in table")}
	m.Results["route -n get -inet 10.4.0.0/16"] = effecttest.Result{Out: []byte(getDefault)}
	failed, logs := RunRoutes(context.Background(), m, tunnel.RoutePlan{Delete: []netip.Prefix{own, other, none, deflt}}, "utun4")
	if len(failed) != 0 {
		t.Errorf("failed = %v", failed)
	}
	wantLogs := []string{
		"leaves the route to 10.2.0.0/16 alone: it goes through utun5, not utun4",
		"leaves the route to 10.4.0.0/16 alone: it goes through en0, not utun4",
	}
	if !slices.Equal(logs, wantLogs) {
		t.Errorf("logs = %q, want %q", logs, wantLogs)
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "run route -q -n delete") && call != "run route -q -n delete -inet 10.1.0.0/16" {
			t.Errorf("deleted by %q", call)
		}
	}
}
