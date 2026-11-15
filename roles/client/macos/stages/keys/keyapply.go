package keys

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/values"
)

// Apply makes what the plan lists, in order. A change but a question is PlanCopy's: a directory
// of the copies with /etc/ws, its mode, or the copy.
func (k *MacKey) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		switch c := c.(type) {
		case askKey:
			err = k.ask(ctx, c.use)
		default:
			err = resource.Group{}.Apply(ctx, step.Plan{Changes: []step.Change{c}})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// refused words the stop where values.yml with a new line would be refused.
func refused(err error) error {
	if errors.Is(err, values.ErrRefused) {
		return fmt.Errorf("%s: ssh has no usable key for ws, as above; mend that, then run the wizard again", stage.Key)
	}
	return err
}
