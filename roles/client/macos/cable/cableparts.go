package cable

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
)

// Part is the cable as one resource, which ws client cable plans and applies.
func (c Cable) Part() resource.Resource { return cablePart{c} }

// cablePart plans the service's setting.
type cablePart struct{ c Cable }

// Plan returns the change, which applies itself.
func (p cablePart) Plan(ctx context.Context) ([]resource.Change, error) {
	plan, err := p.c.plan(ctx)
	if err != nil {
		return nil, err
	}
	return resource.Planned{Changes: plan.Changes}.Plan(ctx)
}
