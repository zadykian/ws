package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/mac/internal/step"
)

// Apply applies the changes of the plan in order, so that root's copy of ws is there when the key
// is made.
func (t *Tunnel) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		switch c := c.(type) {
		case Install:
			err = t.install(ctx, c)
		case Launchctl:
			err = launchctl(ctx, t.M, t.Out, t.sleep, c)
		case KeyMake:
			err = t.makeKey(ctx)
		default:
			err = Apply(ctx, t.M, t.Out, c)
		}
		if err != nil {
			return err
		}
	}
	return t.publish(ctx)
}

func (t *Tunnel) install(ctx context.Context, i Install) error {
	dir := i.Path[:len(i.Path)-len("/ws")]
	if _, err := t.M.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		if _, err := t.M.Run(ctx, "mkdir", "-p", dir); err != nil {
			return fmt.Errorf("cannot make %s: %w", dir, err)
		}
	}
	if err := t.M.WriteFile(i.Path+".new", i.Data, 0o755); err != nil {
		return fmt.Errorf("cannot install %s: %w", i.Path, err)
	}
	if _, err := t.M.Run(ctx, "mv", "-f", i.Path+".new", i.Path); err != nil {
		return fmt.Errorf("cannot install %s: %w", i.Path, err)
	}
	return nil
}
