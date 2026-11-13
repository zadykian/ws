package setup

import (
	"context"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

// plan appends the changes that r plans to cs.
func plan(ctx context.Context, cs []step.Change, r host.Resource) ([]step.Change, error) {
	changes, err := r.Plan(ctx)
	for _, c := range changes {
		cs = append(cs, c)
	}
	return cs, err
}
