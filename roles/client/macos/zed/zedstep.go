package zed

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Step is Zed as a step of resources. A failed cask install is a warning, and the next run
// tries again; where the app's fails, the files wait for it.
func (z *Zed) Step() resource.Lazy {
	return resource.Lazy{Title: "zed", Parts: z.parts}
}

// parts is one part, which reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (z *Zed) parts(context.Context) ([]resource.Resource, error) {
	return []resource.Resource{zedPart{z}}, nil
}

// zedPart plans the app, the font and the files, in that order.
type zedPart struct{ z *Zed }

// Plan returns the changes, each of which applies itself.
func (p zedPart) Plan(ctx context.Context) ([]resource.Change, error) {
	plan, err := p.z.plan(ctx)
	if err != nil {
		return nil, err
	}
	return resource.Planned{Changes: plan.Changes}.Plan(ctx)
}

// Report says how the casks and the files stand.
func (p zedPart) Report(ctx context.Context) ([]step.Item, error) { return p.z.report(ctx) }
