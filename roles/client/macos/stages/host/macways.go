package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/netip"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/routing"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

// MacWays is the Mac's way to ws: the direct way that mac.conf and the Mac's addresses give.
type MacWays struct {
	M effect.Machine
	// Conf is mac.conf's path.
	Conf string
}

// Direct finds the direct way to ws from mac.conf and the Mac's addresses. A failed ifconfig counts
// as no addresses, as in the daemon. A mac.conf that can't be read or parsed leaves no way.
func (w MacWays) Direct(ctx context.Context) (host, port string, ok bool) {
	text, err := w.M.ReadFile(w.Conf)
	if err != nil {
		return "", "", false
	}
	c, errs := conf.Parse(string(text))
	if len(errs) > 0 {
		return "", "", false
	}
	out, err := w.M.Run(ctx, "ifconfig")
	if err != nil {
		out = nil
	}
	d, found := tunnel.DirectWay(routing.ParseIfconfig(string(out), ""), tunnelConf(c), c.RemotePort)
	return d.Host, d.Port, found
}

// tunnelConf takes the keys of mac.conf that a direct way reads. conf checked them already.
func tunnelConf(c conf.Conf) tunnel.Conf {
	t := tunnel.Conf{Remote: c.Remote}
	t.LAN, _ = netip.ParsePrefix(c.LAN)
	t.LANServer, _ = netip.ParseAddr(c.LANAddress)
	return t
}

// Missing says why no way leads to ws: a mac.conf that is missing or refused gives none, and it
// names that rather than ssh. It returns nil where mac.conf is sound.
func (w MacWays) Missing() error {
	_, err := conf.Read(w.M, w.Conf)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("ws is out of reach: no mac.conf at %s; the wizard or ws client macos configure writes it", w.Conf)
	case err != nil:
		return fmt.Errorf("ws is out of reach: mac.conf leaves no way: %w", err)
	}
	return nil
}
