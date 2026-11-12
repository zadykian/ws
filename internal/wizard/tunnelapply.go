package wizard

import (
	"context"
	"errors"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/check"
	"github.com/zadykian/ws/step"
)

// Apply gives ws's wg0 the Mac's key as its peer, where mac.pub holds another or wg0 lacks it. The
// peer op writes the key, then ws server apply --only wireguard runs in a unit of its own. It sets
// WS_PUBLIC_KEY to wg0's key, where it differs, and runs ws mac apply again. Last, it waits for the
// handshake and logs in, or, where the run asked ws's address, leaves that to a later run. It takes
// root's key rather than tunnel.pub, which the user can write, as the key it sends becomes ws's
// peer.
func (t *Tunnel) Apply(ctx context.Context, _ step.Plan) error {
	c, err := t.Ws.want()
	if err != nil {
		return err
	}
	switch err := effect.Validate(ctx, t.M); {
	case errors.Is(err, effect.ErrNoTicket):
		return err
	case err != nil:
		return errors.New(tunnelStage + ": sudo -v failed; the Mac's key is root's, as an admin of the Mac")
	}
	out, err := t.M.RunRoot(ctx, libexec, "mac", "tunnel", "key")
	mac := strings.TrimSpace(string(out))
	if err != nil || !check.Key(mac) {
		return errors.New(tunnelStage + ": root's ws mac tunnel key prints no key: run ws mac apply, which makes both")
	}
	s, _ := t.Ws.status(ctx)
	peered := s.MacKey != mac || !s.Peered
	if peered {
		t.Out.Say("%s: the Mac's key goes to /etc/wireguard/mac.pub on ws, and ws server apply --only "+
			"wireguard runs there in a unit of its own, which gives wg0 that peer", tunnelStage)
		if err := t.Ws.namePeer(ctx, mac); err != nil {
			return err
		}
		var ok bool
		if s, ok = t.Ws.status(ctx); !ok || !s.Peered {
			return errors.New(tunnelStage + ": ws's wg0 lacks the Mac's key after ws server apply --only " +
				"wireguard: mend what it printed, then run the wizard again")
		}
	}
	wg0 := s.WgKey
	if wg0 == "" {
		return errors.New(tunnelStage + ": ws has no wg0 key: mend what it printed, then run the wizard again")
	}
	if wg0 != c.PublicKey {
		file := confFile{t.Conf, tunnelStage, t.M, t.Out}
		if err := file.set(ctx, "WS_PUBLIC_KEY", wg0); err != nil {
			return err
		}
	}
	if _, err := t.MacApply(ctx, false); err != nil {
		return err
	}
	return t.handshake(ctx, c.Target, c.Port, peered && c.PublicKey == wg0)
}
