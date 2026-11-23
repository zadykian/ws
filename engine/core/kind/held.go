package kind

import (
	"context"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/step"
)

// held is a package whose install starts and enables a service that stays off until a later unit
// sets it up. Where the package installs, the service is stopped and disabled, so that a run cut
// short leaves it off at boot too.
type held struct {
	pkg  apt.Package
	u    effect.Units
	unit string
}

type stopHeld struct{ h held }

func (c stopHeld) String() string {
	return "stop and disable " + c.h.unit + ", which the package started"
}

func (stopHeld) Kind() step.ChangeKind { return step.Restart }

func (c stopHeld) Apply(ctx context.Context) error {
	if err := c.h.u.Stop(ctx, c.h.unit); err != nil {
		return err
	}
	return c.h.u.Disable(ctx, c.h.unit)
}

// Plan plans the package's changes, and the stop after its install.
func (h held) Plan(ctx context.Context) ([]resource.Change, error) {
	changes, err := h.pkg.Plan(ctx)
	installs := slices.ContainsFunc(changes, func(c resource.Change) bool { return step.KindOf(c) == step.Install })
	if err != nil || !installs {
		return changes, err
	}
	return append(changes, stopHeld{h}), nil
}
