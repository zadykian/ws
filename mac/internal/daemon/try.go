package daemon

import (
	"context"
	"net/netip"
	"time"

	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/uapi"
)

// keepalive is the peer's persistent keepalive, in seconds.
const keepalive = 25

// try points the peer at c. The routes change first, as a network that holds c and stayed routed
// would take the handshake into the tunnel itself. The peer is removed and added again, as
// wireguard-go handshakes at once with a new peer that has a keepalive. A new endpoint alone would
// wait for the next rekey, up to 2 minutes away.
func (d *Daemon) try(ctx context.Context, c tunnel.Candidate, now time.Time) {
	conf := d.set.Conf
	a := d.allow([]netip.Addr{c.Host})
	d.allowed = a
	d.route(ctx, a)
	if text, err := uapi.Remove(conf.Key); err == nil {
		_ = d.Dev.IpcSet(text) // none to remove at the first try
	}
	end := netip.AddrPortFrom(c.Host, conf.Port)
	text, err := uapi.Peer{Key: conf.Key, Endpoint: end, Allowed: a.IPs(), Keepalive: keepalive}.Add()
	if err == nil {
		err = d.Dev.IpcSet(text)
	}
	if err != nil {
		d.say("cannot set the peer's endpoint to %s", end)
		// The attempt is over before it began, so the next choice moves on.
		d.round = append(d.round, tunnel.Attempt{Candidate: c, At: now.Add(-tunnel.Patience)})
		d.choose(ctx)
		return
	}
	d.round = append(d.round, tunnel.Attempt{Candidate: c, At: now})
	d.later(Check, d.CheckEvery)
}
