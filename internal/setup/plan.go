package setup

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// PlanSteps plans each resource in order, says each change it would make on out as "would ...", and
// returns whether any is due. It changes nothing.
func PlanSteps(ctx context.Context, out effect.Out, steps []resource.Resource) (bool, error) {
	due := false
	for _, r := range steps {
		changes, err := r.Plan(ctx)
		if err != nil {
			return false, err
		}
		for _, c := range changes {
			out.Say("would %s", step.Headline(c))
			due = true
		}
	}
	return due, nil
}
