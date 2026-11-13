// Package resource holds the resources that the steps of any machine's setup share: files, links,
// downloads, packages and units. A resource plans changes that apply themselves. Group joins
// resources into a step.Step.
package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/step"
)

// Change is a step.Change that applies itself.
type Change interface {
	step.Change
	Apply(ctx context.Context) error
}

// Forced is a Change that Group.Apply still applies after another change failed, as a handler runs
// after a failed task. A restart that a watched change calls for is not planned again, since the
// next run finds the watched file right.
type Forced interface {
	Forced() bool
}

// Early is a Change that Group.Plan puts before the changes of all parts. That is for the mark of
// an owed restart, which must be on disk before the change that owes it.
type Early interface {
	Early() bool
}

// Resource is one part of a machine that a step sets up. One that says what it manages is a
// step.Reporter too, and Group joins those.
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

// Report joins what the parts say they manage. A part that says nothing shows by its plan alone.
// A failure keeps what the parts before it said, as a report shows what it can.
func (g Group) Report(ctx context.Context) ([]step.Item, error) {
	var items []step.Item
	for _, part := range g.Parts {
		r, ok := part.(step.Reporter)
		if !ok {
			continue
		}
		found, err := r.Report(ctx)
		items = append(items, found...)
		if err != nil {
			return items, err
		}
	}
	return items, nil
}

// Plan joins the changes of the parts.
func (g Group) Plan(ctx context.Context) (step.Plan, error) {
	var early, rest []step.Change
	for _, part := range g.Parts {
		changes, err := part.Plan(ctx)
		if err != nil {
			return step.Plan{}, err
		}
		for _, c := range changes {
			if e, ok := c.(Early); ok && e.Early() {
				early = append(early, c)
			} else {
				rest = append(rest, c)
			}
		}
	}
	return step.Plan{Changes: append(early, rest...)}, nil
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
			failed = errors.Join(failed, fmt.Errorf("%s: %w", step.Headline(c), err))
		}
	}
	return failed
}
