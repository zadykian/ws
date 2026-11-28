package routing

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	tunnel "github.com/zadykian/ws/macos/roles/tunnel/plan"
)

var (
	corp = netip.MustParsePrefix("10.20.0.0/16")
	lab  = netip.MustParsePrefix("10.20.5.0/24")
)

// getReject is route -n get's answer for a -reject route at n, whoever added it.
func getReject(n netip.Prefix) string {
	a := n.Addr().String()
	return "   route to: " + a + "\ndestination: " + a + "\n       mask: " + net.IP(net.CIDRMask(n.Bits(), 32)).String() +
		"\n    gateway: 127.0.0.1\n  interface: lo0\n      flags: <UP,GATEWAY,REJECT,DONE,STATIC>\n"
}

func TestRunRoutesRejectsAtAYield(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["route -n get -inet 10.20.5.0/24"] = effecttest.Result{Out: []byte(getOn(lab, "utun4"))}
	m.Results["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(getOn(corp, "utun4"))}
	for _, ok := range []string{"delete -inet 10.20.5.0/24", "delete -inet 10.20.0.0/16", "add -inet 10.20.0.0/16 127.0.0.1 -reject"} {
		m.Results["route -q -n "+ok] = effecttest.Result{}
	}
	plan := tunnel.RoutePlan{Delete: []netip.Prefix{lab, corp}}.WithRejects([]netip.Prefix{corp}, []netip.Prefix{corp}, nil)
	ran := RunRoutes(context.Background(), m, plan, "utun4", nil)
	// The reject at the network itself comes once its route is gone, as the two cannot stand at once.
	want := []string{
		"run route -n get -inet 10.20.5.0/24", "run route -q -n delete -inet 10.20.5.0/24",
		"run route -n get -inet 10.20.0.0/16", "run route -q -n delete -inet 10.20.0.0/16",
		"run route -q -n add -inet 10.20.0.0/16 127.0.0.1 -reject",
	}
	if !slices.Equal(m.Calls, want) || !slices.Equal(ran.Rejects, []netip.Prefix{corp}) || len(ran.Logs) != 0 {
		t.Errorf("calls = %q, rejects = %v, logs = %q", m.Calls, ran.Rejects, ran.Logs)
	}
}

func TestRunRoutesRejectsEachNetworkRightAfterItsDelete(t *testing.T) {
	other := netip.MustParsePrefix("172.16.0.0/12")
	m := effecttest.NewMachine()
	m.Results["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(getOn(corp, "utun4"))}
	m.Results["route -n get -inet 172.16.0.0/12"] = effecttest.Result{Out: []byte(getOn(other, "utun4"))}
	for _, ok := range []string{"delete -inet 10.20.0.0/16", "delete -inet 172.16.0.0/12", "add -inet 10.20.0.0/16 127.0.0.1 -reject"} {
		m.Results["route -q -n "+ok] = effecttest.Result{}
	}
	plan := tunnel.RoutePlan{Delete: []netip.Prefix{corp, other}}.WithRejects([]netip.Prefix{corp}, []netip.Prefix{corp}, nil)
	RunRoutes(context.Background(), m, plan, "utun4", nil)
	want := []string{
		"run route -n get -inet 10.20.0.0/16", "run route -q -n delete -inet 10.20.0.0/16",
		"run route -q -n add -inet 10.20.0.0/16 127.0.0.1 -reject",
		"run route -n get -inet 172.16.0.0/12", "run route -q -n delete -inet 172.16.0.0/12",
	}
	if !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q: no network waits through another's delete", m.Calls, want)
	}
}

func TestRunRoutesRejectsAWiderYieldFirst(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["route -n get -inet 10.20.5.0/24"] = effecttest.Result{Out: []byte(getOn(lab, "utun4"))}
	plan := tunnel.RoutePlan{Delete: []netip.Prefix{lab}}.WithRejects([]netip.Prefix{corp}, []netip.Prefix{corp}, nil)
	RunRoutes(context.Background(), m, plan, "utun4", nil)
	if want := "run route -q -n add -inet 10.20.0.0/16 127.0.0.1 -reject"; len(m.Calls) == 0 || m.Calls[0] != want {
		t.Errorf("calls = %q, want %q first", m.Calls, want)
	}
}

func TestRunRoutesLeavesARejectThatStands(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["route -q -n add -inet 10.20.0.0/16 127.0.0.1 -reject"] = effecttest.Result{Err: errors.New("exists")}
	m.Results["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(getReject(corp))}
	ran := RunRoutes(context.Background(), m, tunnel.RoutePlan{Reject: []netip.Prefix{corp}}, "utun4", nil)
	// A daemon that restarts under a yield finds its own reject so, and takes it back as its own.
	if !slices.Equal(ran.Rejects, []netip.Prefix{corp}) || len(ran.Logs) != 0 {
		t.Errorf("rejects = %v, logs = %q; want the reject that stands taken, with no log", ran.Rejects, ran.Logs)
	}
	m.Results["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(getDefault)}
	ran = RunRoutes(context.Background(), m, tunnel.RoutePlan{Reject: []netip.Prefix{corp}}, "utun4", nil)
	want := []string{"cannot add a -reject route to 10.20.0.0/16, so it may go in clear until the other owner routes it"}
	if !slices.Equal(ran.Logs, want) {
		t.Errorf("logs = %q, want %q", ran.Logs, want)
	}
}

func TestRunRoutesTakesBackOverAReject(t *testing.T) {
	m := effecttest.NewMachine()
	add := "route -q -n add -inet 10.20.0.0/16 -interface utun4"
	m.Results[add] = effecttest.Result{Err: errors.New("exists")}
	m.Results["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(getReject(corp))}
	m.Results["route -q -n delete -inet 10.20.0.0/16"] = effecttest.Result{}
	m.After = map[string]map[string]effecttest.Result{"route -q -n delete -inet 10.20.0.0/16": {add: {}}}
	// The reject is the other owner's here: the daemon left none.
	ran := RunRoutes(context.Background(), m, tunnel.RoutePlan{Add: []netip.Prefix{corp}}, "utun4", nil)
	if len(ran.Failed) != 0 || len(ran.Rejects) != 0 || !slices.Contains(m.Calls, "run route -q -n delete -inet 10.20.0.0/16") {
		t.Errorf("failed = %v, rejects = %v, calls = %q", ran.Failed, ran.Rejects, m.Calls)
	}
}

func TestRunRoutesPutsTheRejectBackWhereATakeBackFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["route -q -n add -inet 10.20.0.0/16 -interface utun4"] = effecttest.Result{Err: errors.New("exists")}
	m.Results["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(getReject(corp))}
	m.Results["route -q -n delete -inet 10.20.0.0/16"] = effecttest.Result{}
	m.Results["route -q -n add -inet 10.20.0.0/16 127.0.0.1 -reject"] = effecttest.Result{}
	plan := tunnel.RoutePlan{Add: []netip.Prefix{corp}}.WithRejects(nil, nil, []netip.Prefix{corp})
	ran := RunRoutes(context.Background(), m, plan, "utun4", []netip.Prefix{corp})
	if !slices.Equal(ran.Failed, []netip.Prefix{corp}) || !slices.Equal(ran.Rejects, []netip.Prefix{corp}) {
		t.Errorf("failed = %v, rejects = %v; want both %v", ran.Failed, ran.Rejects, corp)
	}
	want := []string{
		"run route -q -n add -inet 10.20.0.0/16 -interface utun4", "run route -n get -inet 10.20.0.0/16",
		"run route -q -n delete -inet 10.20.0.0/16", "run route -q -n add -inet 10.20.0.0/16 -interface utun4",
		"run route -q -n add -inet 10.20.0.0/16 127.0.0.1 -reject",
	}
	if !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

func TestRunRoutesUnrejects(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(getReject(corp))}
	m.Results["route -q -n add -inet 10.20.5.0/24 -interface utun4"] = effecttest.Result{}
	m.Results["route -q -n delete -inet 10.20.0.0/16"] = effecttest.Result{}
	plan := tunnel.RoutePlan{Add: []netip.Prefix{lab}}.WithRejects(nil, nil, []netip.Prefix{corp})
	ran := RunRoutes(context.Background(), m, plan, "utun4", []netip.Prefix{corp})
	want := []string{
		"run route -q -n add -inet 10.20.5.0/24 -interface utun4",
		"run route -n get -inet 10.20.0.0/16", "run route -q -n delete -inet 10.20.0.0/16",
	}
	if !slices.Equal(m.Calls, want) || len(ran.Rejects) != 0 {
		t.Errorf("calls = %q, rejects = %v", m.Calls, ran.Rejects)
	}
	// A route that took the reject's place is not the daemon's to delete.
	m = effecttest.NewMachine()
	m.Results["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(getOn(corp, "utun5"))}
	ran = RunRoutes(context.Background(), m, tunnel.RoutePlan{Unreject: []netip.Prefix{corp}}, "utun4", []netip.Prefix{corp})
	if slices.Contains(m.Calls, "run route -q -n delete -inet 10.20.0.0/16") || len(ran.Rejects) != 0 {
		t.Errorf("calls = %q, rejects = %v", m.Calls, ran.Rejects)
	}
}
