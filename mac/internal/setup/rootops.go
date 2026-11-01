package setup

import (
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// removeRoot plans the removal of root's file, if it is there.
func removeRoot(m effect.Machine, file string) (step.Change, error) {
	have, err := m.ReadRootFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return RootRemove{file, have}, nil
}

// applyInOrder applies the changes of a plan in order, with launchctl's signal as a change.
func applyInOrder(ctx context.Context, m effect.Machine, out effect.Out, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		if l, ok := c.(Launchctl); ok {
			err = launchctl(ctx, m, out, time.Sleep, l)
		} else {
			err = Apply(ctx, m, out, c)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
