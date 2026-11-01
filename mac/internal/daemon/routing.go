package daemon

import (
	"context"
	"net/netip"
	"slices"

	"github.com/zadykian/ws/mac/internal/tunnel"
)

// allow sorts routes.d by the endpoints hosts, and logs a network once as it goes out. told holds
// the networks logged until keepOut takes them back.
func (d *Daemon) allow(hosts []netip.Addr) tunnel.Allowed {
	a := tunnel.AllowedFor(d.set.Routes, hosts, d.told)
	for _, line := range a.Logs {
		d.say("%s", line)
	}
	for _, n := range a.Out {
		if !slices.Contains(d.told, n) {
			d.told = append(d.told, n)
		}
	}
	return a
}

// keepOut takes the networks of a as those left out, for status, and as those the log has named. The
// log names one again once it went back into the tunnel.
func (d *Daemon) keepOut(a tunnel.Allowed) {
	d.allowed = a
	d.told = slices.Clone(a.Out)
}

// route routes the networks that a wants into the tunnel, and stops routing what it no longer
// holds. A network it cannot route, as one another route already takes, is logged once and tried
// again each time.
func (d *Daemon) route(ctx context.Context, a tunnel.Allowed) {
	plan := tunnel.PlanRoutes(a.Want, d.routed)
	failed, logs := RunRoutes(ctx, d.M, plan, d.Tun)
	var applied []string
	d.routed, d.unroutable, applied = plan.Applied(failed, a.Out, d.unroutable, d.Tun)
	for _, line := range append(logs, applied...) {
		d.say("%s", line)
	}
}

// writeState writes the way in use and the routes for status, through a new file moved over the old
// so that status never reads half of it.
func (d *Daemon) writeState(ctx context.Context) {
	s := tunnel.State{
		Tier: d.inUse.Tier, Interface: d.Tun, Since: d.since, Routes: d.routed, LeftOut: d.allowed.Out,
	}
	if d.inUse.Host.IsValid() {
		s.Endpoint = netip.AddrPortFrom(d.inUse.Host, d.set.Conf.Port).String()
	}
	next := d.StateFile + ".new"
	if err := d.M.WriteFile(next, []byte(s.Encode()), 0o644); err != nil {
		d.say("cannot write %s: %v", next, err)
		return
	}
	if err := d.run(ctx, "mv", "-f", next, d.StateFile); err != nil {
		d.say("cannot write %s: %v", d.StateFile, err)
	}
}

func hostsOf(round []tunnel.Attempt) []netip.Addr {
	var hosts []netip.Addr
	for _, a := range round {
		hosts = append(hosts, a.Candidate.Host)
	}
	return hosts
}
