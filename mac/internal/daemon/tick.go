package daemon

import (
	"context"

	"github.com/zadykian/ws/mac/internal/tunnel"
)

// tick checks that ws answers and looks at the choice again. Decide runs the paths anew where the
// Mac's addresses changed, where no path answered, or where ws missed pings in a row. A tick waits
// while a round is under way, as the round's own checks go on.
func (d *Daemon) tick(ctx context.Context) {
	if len(d.round) > 0 {
		return
	}
	if d.inUse.Host.IsValid() {
		if err := d.run(ctx, "ping", "-c", "1", "-t", "2", "-q", tunnel.Server.String()); err != nil {
			d.misses++
		} else {
			d.misses = 0
		}
	}
	d.choose(ctx)
}

// settleChange runs 2 s after a network change, as one change brings several and DHCP takes a
// moment. It runs the paths again where the Mac's addresses changed, which Decide sees.
func (d *Daemon) settleChange(ctx context.Context) {
	if len(d.round) == 0 {
		d.choose(ctx)
	}
}
