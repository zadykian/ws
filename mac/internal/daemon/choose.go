package daemon

import (
	"context"

	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/uapi"
)

// choose reads the Mac and the peer, asks tunnel.Decide what to do next and does it. A round tries
// the endpoints in order and waits for a handshake from each, so choose runs again after CheckEvery
// while a round is under way.
func (d *Daemon) choose(ctx context.Context) {
	out, err := d.M.Run(ctx, "ifconfig")
	if err != nil {
		// A failed ifconfig counts as no addresses, as the shell did, so that a round goes on.
		d.say("cannot list the Mac's addresses: %v", err)
		out = nil
	}
	ifaces := ParseIfconfig(string(out), d.Tun)
	now := d.Now()
	info := d.peer()
	decide := func() tunnel.Decision {
		return tunnel.Decide(now, ifaces, d.set.Conf, d.resolved, tunnel.Peer{
			InUse: d.inUse, Conf: d.peerConf, Seen: d.seen, Misses: d.misses, Round: d.round,
			Handshake: info.Stats.Handshake, Endpoint: info.Endpoint,
		})
	}
	dec := decide()
	beginning := len(d.round) == 0 && dec.Action != tunnel.Keep
	if beginning {
		// The name is resolved anew at each round, as an address may change.
		if d.set.Conf.Remote != "" {
			d.resolved = Resolve(ctx, d.M, d.set.Conf.Remote)
			dec = decide()
		}
		d.seen, d.peerConf, d.misses = ifaces, d.set.Conf, 0
	}
	if dec.Action == tunnel.Try && dec.Why != "" && dec.Why != tunnel.WhyAddresses && dec.Why != tunnel.WhyNone {
		d.say("%s", dec.Why)
	}
	switch dec.Action {
	case tunnel.Keep:
		if len(d.round) > 0 {
			d.settle(ctx, dec)
		}
	case tunnel.Wait:
		d.later(Check, d.CheckEvery)
	case tunnel.Try:
		d.try(ctx, dec.Candidate, now)
	case tunnel.Fail:
		d.fail(ctx, dec, now)
	}
}

// peer returns what the device reports of the peer, empty where it has none.
func (d *Daemon) peer() uapi.Info {
	text, err := d.Dev.IpcGet()
	if err != nil {
		d.say("cannot read the peer: %v", err)
		return uapi.Info{}
	}
	info, err := uapi.ParseGet(text, d.set.Conf.Key)
	if err != nil {
		d.say("cannot read the peer: %v", err)
	}
	return info
}
