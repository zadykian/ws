package cli

import (
	"context"

	"github.com/zadykian/ws/engine/core/step"
)

// planned is what a stepOf plans.
type planned string

func (c planned) String() string { return string(c) }

// stepOf is a step whose Plan runs plan and finds a change, so that --check names its role.
type stepOf struct {
	name string
	plan func(ctx context.Context) error
}

func (s stepOf) Name() string    { return s.name }
func (s stepOf) WaitsOn() string { return "" }
func (s stepOf) Plan(ctx context.Context) (step.Plan, error) {
	if err := s.plan(ctx); err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: []step.Change{planned("x")}}, nil
}
func (stepOf) Apply(context.Context, step.Plan) error { return nil }
