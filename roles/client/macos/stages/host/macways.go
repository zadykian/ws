package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/netip"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/routing"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
	"github.com/zadykian/ws/roles/client/macos/values"
)

// MacWays is the Mac's way to ws: the direct way that values.yml and the Mac's addresses give.
type MacWays struct {
	M effect.Machine
	// Conf is values.yml's path.
	Conf string
}

// Direct finds the direct way to ws from values.yml and the Mac's addresses. A failed ifconfig counts
// as no addresses, as in the daemon. A values.yml that can't be read or parsed leaves no way.
func (w MacWays) Direct(ctx context.Context) (host, port string, ok bool) {
	text, err := w.M.ReadFile(w.Conf)
	if err != nil {
		return "", "", false
	}
	c, errs := values.Parse(string(text))
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

// tunnelConf takes the keys of values.yml that a direct way reads. values checked them already.
func tunnelConf(c values.Conf) tunnel.Conf {
	t := tunnel.Conf{Remote: c.Remote, Cable: c.CableService != ""}
	t.LAN, _ = netip.ParsePrefix(c.LAN)
	t.LANServer, _ = netip.ParseAddr(c.LANAddress)
	return t
}

// Missing says why no way leads to ws: a values.yml that is missing or refused gives none, and it
// names that rather than ssh. It returns nil where values.yml is sound.
func (w MacWays) Missing() error {
	_, err := values.Read(w.M, w.Conf)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("ws is out of reach: no values.yml at %s; the wizard or ws client macos configure writes it", w.Conf)
	case err != nil:
		return fmt.Errorf("ws is out of reach: the server's values.yml leaves no way: %w", err)
	}
	return nil
}
