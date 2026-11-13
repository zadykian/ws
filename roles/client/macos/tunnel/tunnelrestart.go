package tunnel

import (
	"context"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

// restartOthers plans a kickstart of each other server's loaded daemon where root's copy of ws is
// to change. Every daemon runs that one copy, and a running program keeps the old one. The own
// daemon restarts in daemon(), which needs WS_PUBLIC_KEY. These do not.
func (t *Tunnel) restartOthers(ctx context.Context) ([]step.Change, error) {
	if !t.copied {
		return nil, nil
	}
	names, err := launchd.Installed(t.M)
	if err != nil {
		return nil, err
	}
	var cs []step.Change
	for _, name := range names {
		d := launchd.For(name)
		if name == t.Server {
			continue
		}
		if _, err := t.M.Run(ctx, "launchctl", "print", "system/"+d.Label); err != nil {
			continue
		}
		cs = append(cs, launchd.Launchctl{
			M: t.M, Out: t.Out, Sleep: t.sleep, Label: d.Label,
			Args: []string{"kickstart", "-k", "system/" + d.Label},
		})
	}
	return cs, nil
}
