// Package host holds the resources that the steps of any machine's setup share: files, links,
// downloads, packages and units. A resource plans changes that apply themselves. Group joins
// resources into a step.Step.
package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/step"
)

// Change is a step.Change that applies itself.
type Change interface {
	step.Change
	Apply(ctx context.Context) error
}

// Forced is a Change that Group.Apply still applies after another change failed, as a handler runs
// after a failed task: a restart that a watched change calls for is not planned again, since the
// next run finds the watched file right.
type Forced interface {
	Forced() bool
}

// Resource is one part of a machine that a step sets up.
type Resource interface {
	// Plan reads alone, and returns no change once the resource is as wanted.
	Plan(ctx context.Context) ([]Change, error)
}

// Group is a step.Step of resources. Its changes apply in the order of the parts.
type Group struct {
	Title string
	After string // an earlier step that must be done first, or empty
	Parts []Resource
}

// Name returns the step's name.
func (g Group) Name() string { return g.Title }

// WaitsOn returns the earlier step that must be done first.
func (g Group) WaitsOn() string { return g.After }

// Plan joins the changes of the parts.
func (g Group) Plan(ctx context.Context) (step.Plan, error) {
	var p step.Plan
	for _, part := range g.Parts {
		changes, err := part.Plan(ctx)
		if err != nil {
			return step.Plan{}, err
		}
		for _, c := range changes {
			p.Changes = append(p.Changes, c)
		}
	}
	return p, nil
}

// Apply applies the changes of a plan that Plan made, in order. After the first failure it applies
// the Forced changes alone, and returns the errors.
func (Group) Apply(ctx context.Context, p step.Plan) error {
	var failed error
	for _, c := range p.Changes {
		if f, ok := c.(Forced); failed != nil && (!ok || !f.Forced()) {
			continue
		}
		if err := c.(Change).Apply(ctx); err != nil {
			failed = errors.Join(failed, fmt.Errorf("%s: %w", c, err))
		}
	}
	return failed
}
