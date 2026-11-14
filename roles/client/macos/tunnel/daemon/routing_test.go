package daemon

import (
	"context"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/settings"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
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
	// Once the network went back in, the log names it again as it goes out.
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
	got, err := tunnel.ParseState(string(r.m.Files[r.Paths.State]))
	want := tunnel.State{
		Tier: tunnel.TierLAN, Endpoint: "192.168.1.7:51822", Interface: "utun4", Since: t0,
		Routes: []netip.Prefix{far}, LeftOut: []netip.Prefix{lan},
	}
	if err != nil || got.Encode() != want.Encode() {
		t.Errorf("state = %q, %v; want %q", got.Encode(), err, want.Encode())
	}
	if _, left := r.m.Files[r.Paths.State+".new"]; left || !slices.Contains(r.m.runs, "mv -f "+r.Paths.State+".new "+r.Paths.State) {
		t.Errorf("the new file was not moved over: runs = %q", r.m.runs)
	}
}

func TestWriteStateNone(t *testing.T) {
	r := newRig("")
	r.inUse = tunnel.Candidate{Tier: tunnel.TierNone}
	r.writeState(context.Background())
	if got, _ := tunnel.ParseState(string(r.m.Files[r.Paths.State])); got.Tier != tunnel.TierNone || got.Endpoint != "" {
		t.Errorf("state = %+v, want tier none and no endpoint", got)
	}
}

func TestAllowCountsTheOtherServersEndpoints(t *testing.T) {
	r := newRig("")
	r.m.table["dscacheutil -q host -a name lab.example.org"] = effecttest.Result{Out: []byte("ip_address: 172.16.4.4\n")}
	r.set.Routes = []netip.Prefix{lan, far}
	r.set.Others = []settings.Other{
		{Name: "lab", Conf: tunnel.Conf{LANServer: inLAN, Remote: "lab.example.org"}},
		{Name: "none"},
	}
	r.otherEnds(context.Background())
	if want := []netip.Addr{inLAN, netip.MustParseAddr("172.16.4.4")}; !slices.Equal(r.elsewhere, want) {
		t.Fatalf("elsewhere = %v, want %v", r.elsewhere, want)
	}
	got := r.allow(nil)
	if len(got.Want) != 0 || !slices.Equal(got.Out, []netip.Prefix{lan, far}) {
		t.Errorf("Want = %v, Out = %v, want both left out", got.Want, got.Out)
	}
	if len(r.lines()) != 2 {
		t.Errorf("log = %q, want each network named once", r.lines())
	}
}

func TestChooseCountsTheOtherServersEndpointsAtTheStartOfARound(t *testing.T) {
	r := newRig(confAll)
	r.CheckEvery = time.Hour
	r.m.table["ifconfig"] = effecttest.Result{Out: []byte(ifaces)}
	r.m.table["dscacheutil -q host -a name ws.example.org"] = effecttest.Result{Out: []byte("ip_address: 203.0.113.9\n")}
	r.m.Files["/etc/lab/tunnel.conf"] = []byte("WS_PUBLIC_KEY=" + peerKey + "\nWS_LAN_ADDRESS=192.168.7.7\n")
	r.m.table["ls -1p /etc"] = effecttest.Result{Out: []byte("lab/\nws-tunnel/\n")}
	r.Paths.Root = "/etc"
	if err := r.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.play(Check); err != nil {
		t.Fatal(err)
	}
	if want := []netip.Addr{netip.MustParseAddr("192.168.7.7")}; !slices.Equal(r.elsewhere, want) {
		t.Errorf("elsewhere = %v, want %v", r.elsewhere, want)
	}
}

// heldMachine holds each lookup of the resolver until hold returns, outside fakeMachine's lock,
// as a slow resolver would. hold takes the name looked up.
type heldMachine struct {
	*fakeMachine
	hold func(ctx context.Context, name string) error
}

func (m heldMachine) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "dscacheutil" {
		if err := m.hold(ctx, args[len(args)-1]); err != nil {
			return nil, err
		}
	}
	return m.fakeMachine.Run(ctx, name, args...)
}

func TestOtherEndsLooksUpTheRemotesAtOnce(t *testing.T) {
	r := newRig("")
	r.LookupWait = 10 * time.Second
	// Each lookup answers once both have begun. Lookups in turn would hold the first until
	// LookupWait cuts it off.
	var begun sync.WaitGroup
	begun.Add(2)
	both := make(chan struct{})
	go func() { begun.Wait(); close(both) }()
	r.M = heldMachine{r.m, func(ctx context.Context, _ string) error {
		begun.Done()
		select {
		case <-both:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	r.m.table["dscacheutil -q host -a name a.example.org"] = effecttest.Result{Out: []byte("ip_address: 172.16.4.4\n")}
	r.m.table["dscacheutil -q host -a name b.example.org"] = effecttest.Result{Out: []byte("ip_address: 172.16.5.5\n")}
	r.set.Others = []settings.Other{
		{Name: "a", Conf: tunnel.Conf{Remote: "a.example.org"}},
		{Name: "b", Conf: tunnel.Conf{Remote: "b.example.org"}},
	}
	r.otherEnds(context.Background())
	if want := []netip.Addr{netip.MustParseAddr("172.16.4.4"), netip.MustParseAddr("172.16.5.5")}; !slices.Equal(r.elsewhere, want) {
		t.Errorf("elsewhere = %v, want %v", r.elsewhere, want)
	}
}

func TestOtherEndsCountsALookupCutOffAsUnresolved(t *testing.T) {
	r := newRig("")
	r.LookupWait = 50 * time.Millisecond
	r.M = heldMachine{r.m, func(ctx context.Context, name string) error {
		if name == "slow.example.org" {
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	}}
	r.m.table["dscacheutil -q host -a name lab.example.org"] = effecttest.Result{Out: []byte("ip_address: 172.16.4.4\n")}
	r.set.Others = []settings.Other{
		{Name: "slow", Conf: tunnel.Conf{LANServer: inLAN, Remote: "slow.example.org"}},
		{Name: "lab", Conf: tunnel.Conf{Remote: "lab.example.org"}},
	}
	r.otherEnds(context.Background())
	if want := []netip.Addr{inLAN, netip.MustParseAddr("172.16.4.4")}; !slices.Equal(r.elsewhere, want) {
		t.Errorf("elsewhere = %v, want %v", r.elsewhere, want)
	}
}
