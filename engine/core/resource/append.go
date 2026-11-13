package resource

import (
	"context"

	"github.com/zadykian/ws/engine/core/step"
)

// Append adds the changes that r plans to cs. A failure keeps the changes r planned before it.
func Append(ctx context.Context, cs []step.Change, r Resource) ([]step.Change, error) {
	changes, err := r.Plan(ctx)
	for _, c := range changes {
		cs = append(cs, c)
	}
	return cs, err
}
