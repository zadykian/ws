// Package tunnel holds what Go keeps of the tunnel role, as the commands that its units run under
// ws server tunnel: check, key and wg0. The role sets up wg0, the server's end of a
// WireGuard tunnel for the Mac. See roles/server/ubuntu/tunnel/README.md.
package tunnel

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/server/values"
)

// Check refuses what wg0's files must not act on, before they change. Such are a machine whose
// networkd does not run, and a peer's key file that holds no public key. It names each peer
// left out for want of its key file, and says the server's public key where the key exists.
func Check(ctx context.Context, m effect.Machine, u effect.Units, out effect.Out, t values.Tunnel) error {
	if err := networkdRuns(ctx, u); err != nil {
		return err
	}
	if err := checkPeers(m, out, t); err != nil {
		return err
	}
	public, hasKey, err := serverPublicKey(m)
	switch {
	case err != nil:
		return fmt.Errorf("%s holds no WireGuard private key: %w", keyFile, err)
	case hasKey:
		out.Say("ws's WireGuard public key, for WS_PUBLIC_KEY in a server's values.yml: %s", public)
	default:
		out.Say("The server has no WireGuard key yet: a run makes it and shows its public key.")
	}
	return nil
}

// networkdRuns refuses a machine where systemd-networkd doesn't run, as on Ubuntu Desktop, whose
// NetworkManager does its work instead: nothing would create wg0.
func networkdRuns(ctx context.Context, u effect.Units) error {
	s, err := u.State(ctx, "systemd-networkd.service")
	if err != nil {
		return err
	}
	if !s.Active {
		return errors.New("systemd-networkd is not running, so nothing would create wg0: the role expects networkd, as on Ubuntu Server, not NetworkManager, as on Ubuntu Desktop")
	}
	return nil
}
