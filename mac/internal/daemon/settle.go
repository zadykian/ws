package daemon

import (
	"context"
	"time"

	"github.com/zadykian/ws/mac/internal/tunnel"
)

// settle takes the endpoint that answered as the way in use.
func (d *Daemon) settle(ctx context.Context, dec tunnel.Decision) {
	c := dec.Candidate
	d.keepOut(d.allowed)
	if c != d.inUse {
		d.say("%s", dec.Why)
		d.inUse, d.since = c, d.Now()
	}
	d.round, d.misses = nil, 0
	d.writeState(ctx)
}

// fail leaves the routes in the tunnel, which drops what they carry, rather than let it go out by
// the network the Mac is on. The networks that hold an endpoint tried stay out, for the next
// handshakes.
func (d *Daemon) fail(ctx context.Context, dec tunnel.Decision, now time.Time) {
	a := d.allow(hostsOf(d.round))
	d.route(ctx, a)
	d.keepOut(a)
	if d.inUse.Tier != tunnel.TierNone {
		d.say("%s; trying again 10 s after each round", dec.Why)
		d.inUse, d.since = tunnel.Candidate{Tier: tunnel.TierNone}, now
	}
	d.round, d.misses = nil, 0
	d.writeState(ctx)
}
