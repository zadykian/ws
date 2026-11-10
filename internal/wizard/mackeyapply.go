package wizard

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/step"
)

// Apply makes what the plan lists, in order.
func (k *MacKey) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		switch c := c.(type) {
		case setup.KeyCopy:
			err = setup.Apply(ctx, k.M, k.Out, c)
		case askKey:
			err = k.ask(ctx, c.use)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// refused words the stop where mac.conf with a new line would be refused.
func refused(err error) error {
	if errors.Is(err, errRefused) {
		return fmt.Errorf("%s: ssh has no usable key for ws, as above; mend that, then run the wizard again", macKey)
	}
	return err
}
