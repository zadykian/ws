package cld

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Step is cld as a step of resources.
func (c *Cld) Step() resource.Lazy {
	return resource.Lazy{Title: "cld", Parts: c.parts}
}

// parts is one part, which reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (c *Cld) parts(context.Context) ([]resource.Resource, error) {
	return []resource.Resource{cldPart{c}}, nil
}

// cldPart plans cld's install or update and reports its version.
type cldPart struct{ c *Cld }

// Plan returns the changes, each of which applies itself.
func (p cldPart) Plan(ctx context.Context) ([]resource.Change, error) {
	return resource.Planned{Changes: p.c.plan(ctx).Changes}.Plan(ctx)
}

// Report says whether cld is there, with its version.
func (p cldPart) Report(ctx context.Context) ([]step.Item, error) { return p.c.report(ctx) }
