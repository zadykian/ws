package daemon

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/tunnel"
)

const (
	// ifaces has an address on the cable's subnet and one on the LAN.
	ifaces = "en5: flags=8863<UP,BROADCAST,RUNNING> mtu 1500\n\tinet 10.77.0.2 netmask 0xfffffffc\n" +
		"en0: flags=8863<UP,BROADCAST,RUNNING> mtu 1500\n\tinet 192.168.1.20 netmask 0xffffff00\n"
	confAll = "WS_LAN=192.168.1.0/24\nWS_LAN_ADDRESS=192.168.1.7\nWS_REMOTE=ws.example.org\n"
)

// peerText is what the device reports of the peer, with a handshake at the time at.
func peerText(endpoint string, at time.Time) string {
	return "public_key=" + strings.Repeat("01", 32) + "\nendpoint=" + endpoint +
		"\nlast_handshake_time_sec=" + strconv.FormatInt(at.Unix(), 10) + "\nlast_handshake_time_nsec=0\ntx_bytes=1\nrx_bytes=2\n"
}

// started returns a rig that has started, on the Mac with ifaces and ws's name at 203.0.113.9.
func started(t *testing.T) *rig {
	r := newRig(confAll)
	r.CheckEvery = time.Hour
	r.m.table["ifconfig"] = effecttest.Result{Out: []byte(ifaces)}
	r.m.table["dscacheutil -q host -a name ws.example.org"] = effecttest.Result{Out: []byte("ip_address: 203.0.113.9\n")}
	if err := r.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	return r
}

// added returns the endpoints that the device was given, in order.
func added(r *rig) []string {
	var got []string
	for _, line := range strings.Split(strings.Join(r.dev.sets, "\n"), "\n") {
		if v, ok := strings.CutPrefix(line, "endpoint="); ok {
			got = append(got, v)
		}
	}
	return got
}

func TestChooseTakesTheFirstWayThatAnswers(t *testing.T) {
	r := started(t)
	r.dev.get = peerText("10.77.0.1:51820", t0)
	if err := r.play(Check); err != nil { // the first choice tries the cable, the next finds its handshake
		t.Fatal(err)
	}
	if got := added(r); !slices.Equal(got, []string{"10.77.0.1:51820"}) {
		t.Errorf("endpoints = %q, want the cable's alone", got)
	}
	state, _ := tunnel.ParseState(string(r.m.Files[r.StateFile]))
	if state.Tier != tunnel.TierCable || state.Endpoint != "10.77.0.1:51820" || !state.Since.Equal(t0) {
		t.Errorf("state = %+v", state)
	}
}

func TestChooseMovesOnFromAWayThatIsSilent(t *testing.T) {
	r := started(t)
	ctx := context.Background()
	for range 3 { // the cable, the LAN, the name
		r.choose(ctx)
		r.choose(ctx) // within Patience, it waits
		r.now = r.now.Add(tunnel.Patience)
	}
	r.choose(ctx) // none is left
	if got := added(r); !slices.Equal(got, []string{"10.77.0.1:51820", "192.168.1.7:51820", "203.0.113.9:51820"}) {
		t.Errorf("endpoints = %q", got)
	}
	want := []string{"no path to ws answers, no handshake from 10.77.0.1:51820 on the cable tier, " +
		"no handshake from 192.168.1.7:51820 on the lan tier, " +
		"no handshake from 203.0.113.9:51820 on the remote tier; trying again 10 s after each round"}
	if got := r.lines(); !slices.Equal(got, want) {
		t.Errorf("log = %q, want %q", got, want)
	}
	if state, _ := tunnel.ParseState(string(r.m.Files[r.StateFile])); state.Tier != tunnel.TierNone {
		t.Errorf("state = %+v, want tier none", state)
	}
}
