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
	own, other, none := netip.MustParsePrefix("10.1.0.0/16"), netip.MustParsePrefix("10.2.0.0/16"), netip.MustParsePrefix("10.3.0.0/16")
	deflt, wider, mac := netip.MustParsePrefix("10.4.0.0/16"), netip.MustParsePrefix("10.5.0.0/16"), netip.MustParsePrefix("10.6.0.0/16")
	m := effecttest.NewMachine()
	m.Results["route -n get -inet 10.1.0.0/16"] = effecttest.Result{Out: []byte(getOn(own, "utun4"))}
	m.Results["route -n get -inet 10.2.0.0/16"] = effecttest.Result{Out: []byte(getOn(other, "utun5"))}
	m.Results["route -q -n delete -inet 10.1.0.0/16"] = effecttest.Result{}
	m.Results["route -n get -inet 10.3.0.0/16"] = effecttest.Result{Err: errors.New("not in table")}
	m.Results["route -n get -inet 10.4.0.0/16"] = effecttest.Result{Out: []byte(getDefault)}
	// A wider route through the daemon's own utun answers where the network's own is gone.
	m.Results["route -n get -inet 10.5.0.0/16"] = effecttest.Result{Out: []byte(getOn(netip.MustParsePrefix("10.0.0.0/8"), "utun4"))}
	m.Results["route -n get -inet 10.6.0.0/16"] = effecttest.Result{Out: []byte(getOn(mac, "en0"))}
	plan := tunnel.RoutePlan{Delete: []netip.Prefix{own, other, none, deflt, wider, mac}}
	failed, logs := RunRoutes(context.Background(), m, plan, "utun4")
	if len(failed) != 0 {
		t.Errorf("failed = %v", failed)
	}
	wantLogs := []string{
		"leaves the route to 10.2.0.0/16 alone: it goes through utun5, not utun4",
		"leaves the route to 10.6.0.0/16 alone: it goes through en0, not utun4",
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

func TestRunRoutesDeletesAHostRoute(t *testing.T) {
	// A Mac's answer for a /32 may hold no mask line, or a mask of all ones. Each is the route's own.
	bare, ones := netip.MustParsePrefix("10.7.0.9/32"), netip.MustParsePrefix("10.7.0.10/32")
	m := effecttest.NewMachine()
	m.Results["route -n get -inet 10.7.0.9/32"] = effecttest.Result{
		Out: []byte("   route to: 10.7.0.9\ndestination: 10.7.0.9\n  interface: utun4\n      flags: <UP,HOST,DONE,STATIC>\n"),
	}
	m.Results["route -n get -inet 10.7.0.10/32"] = effecttest.Result{Out: []byte(getOn(ones, "utun4"))}
	m.Results["route -q -n delete -inet 10.7.0.9/32"] = effecttest.Result{}
	m.Results["route -q -n delete -inet 10.7.0.10/32"] = effecttest.Result{}
	failed, logs := RunRoutes(context.Background(), m, tunnel.RoutePlan{Delete: []netip.Prefix{bare, ones}}, "utun4")
	if len(failed) != 0 || len(logs) != 0 {
		t.Errorf("failed = %v, logs = %q", failed, logs)
	}
	for _, n := range []string{"10.7.0.9/32", "10.7.0.10/32"} {
		if !slices.Contains(m.Calls, "run route -q -n delete -inet "+n) {
			t.Errorf("no delete of %s: calls = %q", n, m.Calls)
		}
	}
}
