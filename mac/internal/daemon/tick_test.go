package daemon

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

const ping = "ping -c 1 -t 2 -q 10.99.0.1"

// onTheCable returns a rig whose way in use is the cable.
func onTheCable(t *testing.T) *rig {
	r := started(t)
	r.dev.get = peerText("10.77.0.1:51820", t0)
	if err := r.play(Check); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestTickTriesAgainAfterMissedPings(t *testing.T) {
	r := onTheCable(t)
	ctx := context.Background()
	r.m.table[ping] = effecttest.Result{Err: errTest}
	before := len(r.lines())
	for range 2 {
		r.tick(ctx)
	}
	if got := len(r.lines()); got != before {
		t.Fatalf("log = %q, want no news after 2 misses", r.lines())
	}
	r.tick(ctx)
	want := "ws missed 3 pings in a row; trying the paths again"
	if got := r.lines()[before]; got != want {
		t.Errorf("log = %q, want %q", got, want)
	}
}

func TestTickCountsMissesInARowOnly(t *testing.T) {
	r := onTheCable(t)
	ctx := context.Background()
	r.m.table[ping] = effecttest.Result{Err: errTest}
	r.tick(ctx)
	r.tick(ctx)
	delete(r.m.table, ping)
	r.tick(ctx)
	r.m.table[ping] = effecttest.Result{Err: errTest}
	r.tick(ctx)
	r.tick(ctx)
	if r.misses != 2 || slices.ContainsFunc(r.lines(), func(l string) bool { return strings.HasPrefix(l, "ws missed") }) {
		t.Errorf("misses = %d, log = %q", r.misses, r.lines())
	}
}

func TestTickWaitsForARound(t *testing.T) {
	r := started(t)
	ctx := context.Background()
	r.choose(ctx) // tries the cable, which has no handshake yet
	runs := len(r.m.runs)
	r.tick(ctx)
	if len(r.m.runs) != runs {
		t.Errorf("commands = %q, want none in a round", r.m.runs[runs:])
	}
}

func TestSettleChangeTriesAgainWhereTheAddressesChanged(t *testing.T) {
	r := onTheCable(t)
	ctx := context.Background()
	r.settleChange(ctx)
	if got := added(r); len(got) != 1 {
		t.Fatalf("endpoints = %q, want the way kept where nothing changed", got)
	}
	r.m.table["ifconfig"] = effecttest.Result{Out: []byte(strings.Replace(ifaces, "192.168.1.20", "192.168.1.21", 1))}
	r.settleChange(ctx)
	if got := added(r); len(got) != 2 {
		t.Errorf("endpoints = %q, want the paths tried again", got)
	}
}
