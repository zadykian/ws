package daemon

import (
	"context"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	tunnel "github.com/zadykian/ws/macos/roles/tunnel/plan"
)

var (
	corp   = netip.MustParsePrefix("10.20.0.0/16")
	office = []netip.Prefix{corp, far}
)

// ownCorp is route -n get's answer for corp through the daemon's utun.
const ownCorp = "   route to: 10.20.0.0\ndestination: 10.20.0.0\n       mask: 255.255.0.0\n" +
	"  interface: utun4\n      flags: <UP,DONE,STATIC,PRCLONING>\n"

// withOffice gives the rig's routes.d the networks of office.
func withOffice(r *rig) {
	r.m.Files[dir+"/routes.d/office"] = []byte("10.20.0.0/16\n172.16.0.0/12\n")
	r.m.table["ls -1 "+dir+"/routes.d"] = effecttest.Result{Out: []byte("office\n")}
}

// stateOf writes the rig's state and reads it back.
func stateOf(t *testing.T, r *rig) tunnel.State {
	r.writeState(context.Background())
	s, err := tunnel.ParseState(string(r.m.Files[r.Paths.State]))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestADaemonThatStartsWithAYieldLeavesItsNetworksOut(t *testing.T) {
	r := newRig(confAll)
	r.CheckEvery = time.Hour
	r.m.table["ifconfig"] = effecttest.Result{Out: []byte(ifaces)}
	withOffice(r)
	r.m.Files[paths.Yield] = []byte("10.20.0.0/16\n")
	r.dev.get = peerText("10.77.0.1:51820", t0)
	if err := r.play(Check); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(r.m.runs, "route -q -n add -inet 10.20.0.0/16 -interface utun4") {
		t.Errorf("routed the yielded network: commands = %q", r.m.runs)
	}
	if got := r.dev.sets[len(r.dev.sets)-1]; strings.Contains(got, "10.20.0.0/16") || !strings.Contains(got, "172.16.0.0/12") {
		t.Errorf("last set = %q, want AllowedIPs without the yielded network", got)
	}
	if want := "leaves 10.20.0.0/16 out, as it yields 10.20.0.0/16 to another owner"; !slices.Contains(r.lines(), want) {
		t.Errorf("log = %q, want %q", r.lines(), want)
	}
	if s := stateOf(t, r); !slices.Equal(s.Routes, []netip.Prefix{far}) || !slices.Equal(s.Yielded, []netip.Prefix{corp}) {
		t.Errorf("state = %q, want routes %v and yielded %v", s.Encode(), far, corp)
	}
}

func TestReloadYieldsANetworkAndTakesItBack(t *testing.T) {
	r := onTheCable(t)
	withOffice(r)
	ctx := context.Background()
	r.reload(ctx)
	r.m.table["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(ownCorp)}
	r.m.Files[paths.Yield] = []byte("10.20.0.0/16\n")
	r.reload(ctx)
	if !slices.Contains(r.m.runs, "route -q -n delete -inet 10.20.0.0/16") {
		t.Errorf("kept the yielded network's route: commands = %q", r.m.runs)
	}
	if s := stateOf(t, r); !slices.Equal(s.Routes, []netip.Prefix{far}) || !slices.Equal(s.Yielded, []netip.Prefix{corp}) {
		t.Errorf("state with the yield = %q", s.Encode())
	}
	delete(r.m.Files, paths.Yield)
	r.m.runs = nil
	r.reload(ctx)
	if !slices.Contains(r.m.runs, "route -q -n add -inet 10.20.0.0/16 -interface utun4") {
		t.Errorf("did not take the network back: commands = %q", r.m.runs)
	}
	if s := stateOf(t, r); !slices.Equal(s.Routes, office) || len(s.Yielded) != 0 {
		t.Errorf("state once the yield is gone = %q", s.Encode())
	}
	if got := r.dev.sets[len(r.dev.sets)-1]; !strings.Contains(got, "10.20.0.0/16") {
		t.Errorf("last set = %q, want AllowedIPs with the network again", got)
	}
}

// rejectCorp is route -n get's answer for a -reject route at corp, and rejectAt the line that adds one.
const (
	rejectCorp = "   route to: 10.20.0.0\ndestination: 10.20.0.0\n       mask: 255.255.0.0\n    gateway: 127.0.0.1\n" +
		"  interface: lo0\n      flags: <UP,GATEWAY,REJECT,DONE,STATIC>\n"
	rejectAt = "route -q -n add -inet 10.20.0.0/16 127.0.0.1 -reject"
)

func TestAYieldLeavesARejectThatATakeBackKeepsWhereItFails(t *testing.T) {
	r := onTheCable(t)
	withOffice(r)
	ctx := context.Background()
	r.reload(ctx)
	r.m.table["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(ownCorp)}
	r.m.Files[paths.Yield] = []byte("10.20.0.0/16\n")
	r.m.runs = nil
	r.reload(ctx)
	del := slices.Index(r.m.runs, "route -q -n delete -inet 10.20.0.0/16")
	if rej := slices.Index(r.m.runs, rejectAt); del < 0 || rej < del {
		t.Fatalf("want a reject once the route is gone: commands = %q", r.m.runs)
	}
	// The reject stands in the route's place, and the add of the take-back fails over it too.
	add := "route -q -n add -inet 10.20.0.0/16 -interface utun4"
	r.m.table["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(rejectCorp)}
	r.m.table[add] = effecttest.Result{Err: errTest}
	delete(r.m.Files, paths.Yield)
	r.m.runs = nil
	r.reload(ctx)
	want := []string{add, "route -n get -inet 10.20.0.0/16", "route -q -n delete -inet 10.20.0.0/16", add, rejectAt}
	if i := slices.Index(r.m.runs, add); i < 0 || !slices.Equal(r.m.runs[i:min(i+len(want), len(r.m.runs))], want) {
		t.Errorf("commands = %q, want %q: the reject back in place", r.m.runs, want)
	}
	if n := slices.Index(r.m.runs, rejectAt); slices.Contains(r.m.runs[n+1:], "route -q -n delete -inet 10.20.0.0/16") {
		t.Errorf("removed the reject that it put back: commands = %q", r.m.runs)
	}
	if s := stateOf(t, r); !slices.Equal(s.Routes, []netip.Prefix{far}) || !slices.Equal(r.rejects, []netip.Prefix{corp}) {
		t.Errorf("state = %q, rejects = %v", s.Encode(), r.rejects)
	}
}

func TestAMovedYieldMovesTheReject(t *testing.T) {
	r := onTheCable(t)
	withOffice(r)
	ctx := context.Background()
	r.reload(ctx)
	r.m.table["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(ownCorp)}
	r.m.Files[paths.Yield] = []byte("10.20.0.0/16\n")
	r.reload(ctx)
	// The network is out of the routes already, so only the held prefix says where its reject goes.
	r.m.table["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(rejectCorp)}
	r.m.Files[paths.Yield] = []byte("10.0.0.0/8\n")
	r.m.runs = nil
	r.reload(ctx)
	wide, unreject := "route -q -n add -inet 10.0.0.0/8 127.0.0.1 -reject", "route -q -n delete -inet 10.20.0.0/16"
	if add, del := slices.Index(r.m.runs, wide), slices.Index(r.m.runs, unreject); add < 0 || del < add {
		t.Errorf("commands = %q, want the reject at 10.0.0.0/8 before the one at 10.20.0.0/16 goes", r.m.runs)
	}
	if want := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}; !slices.Equal(r.rejects, want) {
		t.Errorf("rejects = %v, want %v", r.rejects, want)
	}
}

func TestADaemonThatRestartsUnderAYieldTakesTheRejectThatStands(t *testing.T) {
	r := newRig(confAll)
	r.CheckEvery = time.Hour
	r.m.table["ifconfig"] = effecttest.Result{Out: []byte(ifaces)}
	withOffice(r)
	r.m.Files[paths.Yield] = []byte("10.20.0.0/16\n")
	r.m.table[rejectAt] = effecttest.Result{Err: errTest}
	r.m.table["route -n get -inet 10.20.0.0/16"] = effecttest.Result{Out: []byte(rejectCorp)}
	r.dev.get = peerText("10.77.0.1:51820", t0)
	if err := r.play(Check); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.rejects, []netip.Prefix{corp}) {
		t.Errorf("rejects = %v, want %v: the reject of the daemon before it", r.rejects, corp)
	}
	delete(r.m.Files, paths.Yield)
	r.m.runs = nil
	r.reload(context.Background())
	if !slices.Contains(r.m.runs, "route -q -n delete -inet 10.20.0.0/16") {
		t.Errorf("commands = %q, want the reject removed at the take-back", r.m.runs)
	}
}
