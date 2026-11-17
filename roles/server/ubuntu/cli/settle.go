package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// errDue is the failure of a command's --check that finds a change to make, which a run state
// takes for a run to do.
var errDue = errors.New("to do")

// settle plans each resource of rs, in order. With check, it fails with errDue and every change's
// headline where one is due. Otherwise it applies each resource's changes before it plans the next,
// and says each on out.
func settle(ctx context.Context, out effect.Out, rs []resource.Resource, check bool) error {
	var due []string
	for _, r := range rs {
		changes, err := r.Plan(ctx)
		if err != nil {
			return err
		}
		for _, c := range changes {
			if check {
				due = append(due, step.Headline(c))
				continue
			}
			if err := c.Apply(ctx); err != nil {
				return err
			}
			out.Say("%s", step.Headline(c))
		}
	}
	if len(due) > 0 {
		return fmt.Errorf("%w: %s", errDue, strings.Join(due, "; "))
	}
	return nil
}
