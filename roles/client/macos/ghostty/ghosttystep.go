package ghostty

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Step is ghostty as a step of resources. A failed cask install is a warning, and the next run
// tries again; where the app's fails, the files wait for it.
func (g *Ghostty) Step() resource.Lazy {
	return resource.Lazy{Title: "ghostty", Parts: g.parts}
}

// parts is one part, which reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (g *Ghostty) parts(context.Context) ([]resource.Resource, error) {
	return []resource.Resource{ghosttyPart{g}}, nil
}

// ghosttyPart plans the app, the font and the files, in that order.
type ghosttyPart struct{ g *Ghostty }

// Plan returns the changes, each of which applies itself.
func (p ghosttyPart) Plan(ctx context.Context) ([]resource.Change, error) {
	plan, err := p.g.plan(ctx)
	if err != nil {
		return nil, err
	}
	return resource.Planned{Changes: plan.Changes}.Plan(ctx)
}

// Report says how the casks, ws.ghostty and the line that reads it stand.
func (p ghosttyPart) Report(ctx context.Context) ([]step.Item, error) { return p.g.report(ctx) }
