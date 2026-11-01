package step_test

import (
	"context"

	"github.com/zadykian/ws/mac/internal/step"
)

type change string

func (c change) String() string { return string(c) }

// fake is a step whose plan holds todo until Apply empties it.
type fake struct {
	name     string
	todo     []step.Change
	waitsOn  string
	sticky   bool // Apply changes nothing
	planErr  error
	applyErr error
	log      *[]string
}

func (f *fake) Name() string    { return f.name }
func (f *fake) WaitsOn() string { return f.waitsOn }

func (f *fake) Plan(context.Context) (step.Plan, error) {
	*f.log = append(*f.log, "plan "+f.name)
	return step.Plan{Changes: f.todo}, f.planErr
}

func (f *fake) Apply(context.Context, step.Plan) error {
	*f.log = append(*f.log, "apply "+f.name)
	if !f.sticky {
		f.todo = nil
	}
	return f.applyErr
}
