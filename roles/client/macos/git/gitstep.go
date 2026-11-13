package git

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Step is git as a step of resources. It plans alone where mac.conf gives no git name.
func (g *Git) Step() resource.Lazy {
	return resource.Lazy{Title: "git", Parts: g.parts}
}

// parts is one part, which reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (g *Git) parts(context.Context) ([]resource.Resource, error) {
	if g.Conf.GitName == "" {
		return nil, nil
	}
	return []resource.Resource{gitPart{g}}, nil
}

// gitPart plans git's files, the include, gh and the key's registration, in that order.
type gitPart struct{ g *Git }

// Plan returns the changes, each of which applies itself.
func (p gitPart) Plan(ctx context.Context) ([]resource.Change, error) {
	plan, err := p.g.Plan(ctx)
	if err != nil {
		return nil, err
	}
	return resource.Planned{Changes: plan.Changes}.Plan(ctx)
}

// Report says how gh, the key, the files and the include stand.
func (p gitPart) Report(ctx context.Context) ([]step.Item, error) { return p.g.Report(ctx) }
