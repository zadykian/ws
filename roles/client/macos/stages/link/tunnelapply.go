package link

import (
	"context"
	"errors"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// Apply gives ws's wg0 the Mac's key as its peer, where mac.pub holds another or wg0 lacks it. The
// peer op writes the key, then ws server ubuntu apply --only wireguard runs in a unit of its own.
// It sets WS_PUBLIC_KEY to wg0's key, where it differs, and runs ws client macos apply again. Last,
// it waits for the handshake and logs in, or, where the run asked ws's address, leaves that to a
// later run. It takes root's key rather than tunnel.pub, which the user can write, as the key it
// sends becomes ws's peer.
func (t *Tunnel) Apply(ctx context.Context, _ step.Plan) error {
	switch err := local.Validate(ctx, t.M); {
	case errors.Is(err, local.ErrNoTicket):
		return err
	case err != nil:
		return errors.New(stage.Tunnel + ": sudo -v failed; the Mac's key is root's, as an admin of the Mac")
	}
	c, err := conf.Read(t.M, t.Conf)
	if err != nil {
		return err
	}
	out, err := t.M.RunRoot(ctx, libexec, "client", "macos", "tunnel", "key")
	mac := strings.TrimSpace(string(out))
	if err != nil || !check.Key(mac) {
		return errors.New(stage.Tunnel + ": root's ws client macos tunnel key prints no key: run ws client macos apply, which makes both")
	}
	s, _ := t.Ws.Status(ctx)
	peered := s.MacKey != mac || !s.Peered
	if peered {
		t.Out.Say("%s: the Mac's key goes to /etc/wireguard/mac.pub on ws, and ws server ubuntu apply --only "+
			"wireguard runs there in a unit of its own, which gives wg0 that peer", stage.Tunnel)
		if err := t.Ws.NamePeer(ctx, mac); err != nil {
			return err
		}
		var ok bool
		if s, ok = t.Ws.Status(ctx); !ok || !s.Peered {
			return errors.New(stage.Tunnel + ": ws's wg0 lacks the Mac's key after ws server ubuntu apply --only " +
				"wireguard: mend what it printed, then run the wizard again")
		}
	}
	wg0 := s.WgKey
	if wg0 == "" {
		return errors.New(stage.Tunnel + ": ws has no wg0 key: mend what it printed, then run the wizard again")
	}
	if wg0 != c.PublicKey {
		file := conf.File{Path: t.Conf, Stage: stage.Tunnel, M: t.M, Out: t.Out}
		if err := file.Set(ctx, "WS_PUBLIC_KEY", wg0); err != nil {
			return err
		}
	}
	if _, err := t.MacApply(ctx, false); err != nil {
		return err
	}
	return t.handshake(ctx, c, peered && c.PublicKey == wg0)
}
