package tunnel

import (
	"context"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

// Apply applies the changes of the plan in order, so that root's copy of ws is there when the key
// is made.
func (t *Tunnel) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		switch c := c.(type) {
		case Install:
			err = c.Apply(ctx)
		case launchd.Launchctl:
			err = c.Apply(ctx)
		case KeyMake:
			err = c.Apply(ctx)
		default:
			err = change.Apply(ctx, t.M, t.Out, c)
		}
		if err != nil {
			return err
		}
	}
	return t.publish(ctx)
}
