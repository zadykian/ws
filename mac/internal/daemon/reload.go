package daemon

import (
	"context"

	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/uapi"
)

// reload reads tunnel.conf and routes.d again, on SIGHUP. A new key or path has the paths tried
// again. Otherwise it changes the peer's AllowedIPs and the routes in place, which keeps the
// session. The routes stay clear of the endpoints of the last round.
func (d *Daemon) reload(ctx context.Context) {
	set, err := ReadSettings(ctx, d.M, d.Dir)
	if err != nil {
		d.say("%v; keeps the settings it had", err)
		set.Conf = d.set.Conf
	}
	for _, w := range set.Warnings {
		d.say("%s", w)
	}
	old := d.set.Conf
	d.set = set
	if set.Conf != old {
		if set.Conf.Key != old.Key {
			if text, err := uapi.Remove(old.Key); err == nil {
				_ = d.Dev.IpcSet(text) // none to remove before the first try
			}
		}
		// A round that is under way ends first, and the next tick sees the new settings.
		if len(d.round) == 0 {
			d.choose(ctx)
		}
		return
	}
	hosts := d.ends
	if len(d.round) > 0 {
		hosts = hostsOf(d.round)
	}
	a := d.allow(hosts)
	// The peer exists once a round has tried an endpoint, whether it answered or not.
	if len(d.round) > 0 || len(d.ends) > 0 {
		text, err := uapi.SetAllowed(set.Conf.Key, a.IPs())
		if err == nil {
			err = d.Dev.IpcSet(text)
		}
		if err != nil {
			d.say("cannot set the peer's AllowedIPs")
		}
	}
	d.route(ctx, a)
	d.keepOut(a)
	routed := tunnel.JoinNets(d.routed)
	if routed == "" {
		routed = "none"
	}
	d.say("read its settings again; routes: %s", routed)
	d.writeState(ctx)
}
