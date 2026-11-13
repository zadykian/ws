package cable

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Step is the cable as a step of resources.
func (c Cable) Step() resource.Lazy {
	return resource.Lazy{Title: "cable", Parts: c.parts}
}

// parts is one part, which reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (c Cable) parts(context.Context) ([]resource.Resource, error) {
	return []resource.Resource{cablePart{c}}, nil
}

// cablePart plans the service's setting and reports how the service stands.
type cablePart struct{ c Cable }

// Plan returns the change, which applies itself.
func (p cablePart) Plan(ctx context.Context) ([]resource.Change, error) {
	plan, err := p.c.plan(ctx)
	if err != nil {
		return nil, err
	}
	return resource.Planned{Changes: plan.Changes}.Plan(ctx)
}

// Report says how the service stands.
func (p cablePart) Report(ctx context.Context) ([]step.Item, error) { return p.c.report(ctx) }
