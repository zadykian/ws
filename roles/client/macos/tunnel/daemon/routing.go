package daemon

import (
	"context"
	"net/netip"
	"slices"
	"sync"

	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/routing"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

// otherEnds sets the endpoints of the other servers: each one's LAN address, and its remote as
// the system's resolver gives it now. A network that holds one would send that server's own
// packets into this server's tunnel. The loop waits on the lookups, so they run at once, under
// LookupWait for all: a lookup cut off counts as unresolved, as a failed one does.
func (d *Daemon) otherEnds(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, d.LookupWait)
	defer cancel()
	remotes := make([]netip.Addr, len(d.set.Others))
	var lookups sync.WaitGroup
	for i, o := range d.set.Others {
		if o.Conf.Remote != "" {
			lookups.Go(func() { remotes[i] = routing.Resolve(ctx, d.M, o.Conf.Remote) })
		}
	}
	lookups.Wait()
	d.elsewhere = nil
	for i, o := range d.set.Others {
		if o.Conf.LANServer.IsValid() {
			d.elsewhere = append(d.elsewhere, o.Conf.LANServer)
		}
		if remotes[i].IsValid() {
			d.elsewhere = append(d.elsewhere, remotes[i])
		}
	}
}

// allow sorts routes.d by the endpoints hosts and those of the other servers, and logs a network once as it goes out. told holds
// the networks logged until keepOut takes them back.
func (d *Daemon) allow(hosts []netip.Addr) tunnel.Allowed {
	a := tunnel.AllowedFor(d.set.Routes, append(slices.Clone(hosts), d.elsewhere...), d.told)
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
	plan := tunnel.Routes(a.Want, d.routed)
	failed, logs := routing.RunRoutes(ctx, d.M, plan, d.Tun)
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
	next := d.Paths.State + ".new"
	if err := d.M.WriteFile(next, []byte(s.Encode()), 0o644); err != nil {
		d.say("cannot write %s: %v", next, err)
		return
	}
	if err := d.run(ctx, "mv", "-f", next, d.Paths.State); err != nil {
		d.say("cannot write %s: %v", d.Paths.State, err)
	}
}

func hostsOf(round []tunnel.Attempt) []netip.Addr {
	var hosts []netip.Addr
	for _, a := range round {
		hosts = append(hosts, a.Candidate.Host)
	}
	return hosts
}
