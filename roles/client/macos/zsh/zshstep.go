package zsh

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Step is zsh as a step of resources.
func (z *Zsh) Step() resource.Lazy {
	return resource.Lazy{Title: "zsh", Parts: z.parts}
}

// parts is one part, which reads the Mac when it plans or reports, so that a report warns of
// nothing.
func (z *Zsh) parts(context.Context) ([]resource.Resource, error) {
	return []resource.Resource{zshPart{z}}, nil
}

// zshPart plans the directories, the hook and the line that sources it, in that order.
type zshPart struct{ z *Zsh }

// Plan returns the changes, each of which applies itself.
func (p zshPart) Plan(ctx context.Context) ([]resource.Change, error) {
	plan, err := p.z.plan(ctx)
	if err != nil {
		return nil, err
	}
	return resource.Planned{Changes: plan.Changes}.Plan(ctx)
}

// Report says how the hook and the line stand.
func (p zshPart) Report(ctx context.Context) ([]step.Item, error) { return p.z.report(ctx) }
