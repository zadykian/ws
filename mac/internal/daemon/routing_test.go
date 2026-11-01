package daemon

import (
	"context"
	"net/netip"
	"slices"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/tunnel"
)

var (
	lan   = netip.MustParsePrefix("192.168.1.0/24")
	far   = netip.MustParsePrefix("172.16.0.0/12")
	inLAN = netip.MustParseAddr("192.168.1.7")
)

func TestAllowLogsAnOutNetworkOnce(t *testing.T) {
	r := newRig("")
	r.set.Routes = []netip.Prefix{lan, far}
	got := r.allow([]netip.Addr{inLAN})
	if !slices.Equal(got.Want, []netip.Prefix{far}) || !slices.Equal(got.Out, []netip.Prefix{lan}) {
		t.Errorf("Want = %v, Out = %v", got.Want, got.Out)
	}
	r.allow([]netip.Addr{inLAN})
	want := []string{"leaves 192.168.1.0/24 out of the tunnel while it holds the endpoint, 192.168.1.7"}
	if !slices.Equal(r.lines(), want) {
		t.Errorf("log = %q, want %q", r.lines(), want)
	}
	// Once the network went back in, it is logged again as it goes out.
	r.keepOut(r.allow(nil))
	r.allow([]netip.Addr{inLAN})
	if len(r.lines()) != 2 {
		t.Errorf("log = %q, want the network named twice", r.lines())
	}
}

func TestRouteLogsAFailureOnce(t *testing.T) {
	r := newRig("")
	r.m.table["route -q -n add -inet 192.168.1.0/24 -interface utun4"] = effecttest.Result{Err: errTest}
	for range 2 {
		r.route(context.Background(), tunnel.Allowed{Want: []netip.Prefix{lan, far}})
	}
	if !slices.Equal(r.routed, []netip.Prefix{far}) || !slices.Equal(r.unroutable, []netip.Prefix{lan}) {
		t.Errorf("routed = %v, unroutable = %v", r.routed, r.unroutable)
	}
	want := []string{"cannot route 192.168.1.0/24 into utun4, as another route to it may be in the way"}
	if !slices.Equal(r.lines(), want) {
		t.Errorf("log = %q, want %q", r.lines(), want)
	}
}

func TestWriteState(t *testing.T) {
	r := newRig("WS_PORT=51822\n")
	if err := r.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.inUse = tunnel.Candidate{Tier: tunnel.TierLAN, Host: inLAN}
	r.since, r.routed = t0, []netip.Prefix{far}
	r.allowed.Out = []netip.Prefix{lan}
	r.writeState(context.Background())
	got, err := tunnel.ParseState(string(r.m.Files[r.StateFile]))
	want := tunnel.State{
		Tier: tunnel.TierLAN, Endpoint: "192.168.1.7:51822", Interface: "utun4", Since: t0,
		Routes: []netip.Prefix{far}, LeftOut: []netip.Prefix{lan},
	}
	if err != nil || got.Encode() != want.Encode() {
		t.Errorf("state = %q, %v; want %q", got.Encode(), err, want.Encode())
	}
	if _, left := r.m.Files[r.StateFile+".new"]; left || !slices.Contains(r.m.runs, "mv -f "+r.StateFile+".new "+r.StateFile) {
		t.Errorf("the new file was not moved over: runs = %q", r.m.runs)
	}
}

func TestWriteStateNone(t *testing.T) {
	r := newRig("")
	r.inUse = tunnel.Candidate{Tier: tunnel.TierNone}
	r.writeState(context.Background())
	if got, _ := tunnel.ParseState(string(r.m.Files[r.StateFile])); got.Tier != tunnel.TierNone || got.Endpoint != "" {
		t.Errorf("state = %+v, want tier none and no endpoint", got)
	}
}
