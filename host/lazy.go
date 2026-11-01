package host

import (
	"context"

	"github.com/zadykian/ws/step"
)

// Lazy is a Group whose parts are made as it plans, from what the machine shows then: a value
// that an earlier step settles, such as the release a source names or the size of a disk. Its
// plan and its apply see the same parts, as Plan makes them again each time.
type Lazy struct {
	Title string
	After string // an earlier step that must be done first, or empty
	Parts func(ctx context.Context) ([]Resource, error)
}

// Name returns the step's name.
func (l Lazy) Name() string { return l.Title }

// WaitsOn returns the earlier step that must be done first.
func (l Lazy) WaitsOn() string { return l.After }

// Plan makes the parts and plans them as a Group does.
func (l Lazy) Plan(ctx context.Context) (step.Plan, error) {
	parts, err := l.Parts(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	return Group{Title: l.Title, Parts: parts}.Plan(ctx)
}

// Apply applies the plan as a Group does.
func (l Lazy) Apply(ctx context.Context, p step.Plan) error { return Group{}.Apply(ctx, p) }
