package keys

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// Apply makes what the plan lists, in order. A change but a question is PlanCopy's: a directory
// of the copies, its mode, or the copy.
func (k *MacKey) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		switch c := c.(type) {
		case askKey:
			err = k.ask(ctx, c.use)
		default:
			err = change.Apply(ctx, k.M, k.Out, c)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// refused words the stop where mac.conf with a new line would be refused.
func refused(err error) error {
	if errors.Is(err, conf.ErrRefused) {
		return fmt.Errorf("%s: ssh has no usable key for ws, as above; mend that, then run the wizard again", stage.Key)
	}
	return err
}
