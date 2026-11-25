package tunnel

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/values"
)

// GuestCheck refuses a host whose networkd does not run, or a peer's key file of a guest's tunnel
// that holds no key, before the guests' interfaces change. It names each peer left out for want of
// its key file, and says the public key of each interface that has one.
func GuestCheck(ctx context.Context, m effect.Machine, u effect.Units, out effect.Out, tunnels []values.GuestTunnel) error {
	if err := networkdRuns(ctx, u); err != nil {
		return err
	}
	for _, t := range tunnels {
		if err := checkPeers(m, out, t.Link(), t.PeersDir(), t.Peers); err != nil {
			return err
		}
		public, hasKey, err := publicKeyOf(m, t.KeyFile())
		if err != nil {
			return err
		}
		if hasKey {
			out.Say("%s's WireGuard public key, for the peer of %s on the Mac: %s", t.Link(), t.Guest.Name, public)
		}
	}
	return nil
}

// GuestKeys returns the keys that each guest's interface reads, made where they are missing.
func GuestKeys(m effect.Machine, out effect.Out, tunnels []values.GuestTunnel) []resource.Resource {
	var keys []resource.Resource
	for _, t := range tunnels {
		keys = append(keys, Key{M: m, Out: out, File: t.KeyFile()})
	}
	return keys
}
