package tunnel

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Apply applies the changes of the plan in order, so that root's copy of ws is there when the key
// is made.
func (t *Tunnel) Apply(ctx context.Context, p step.Plan) error {
	if err := (resource.Group{}).Apply(ctx, p); err != nil {
		return err
	}
	return t.publish(ctx)
}
