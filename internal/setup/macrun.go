package setup

import (
	"context"

	"github.com/zadykian/ws/step"
)

// run runs steps under o, then what each leaves for the end of a run that passed. A step left out
// leaves nothing, as it applied nothing. It reports what step.RunWith does.
func run(ctx context.Context, steps []step.Step, o step.Options) (bool, error) {
	changed, err := step.RunWith(ctx, steps, o)
	for _, s := range steps {
		left := o.LeftOut != nil && o.LeftOut(o.Parent, s.Name())
		if f, ok := s.(interface{ Finish(context.Context) error }); ok && err == nil && !left {
			err = f.Finish(ctx)
		}
	}
	return changed, err
}
