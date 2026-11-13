package engine

import (
	"context"
	"io"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

// pick opens the list of run's steps, scanned by a check. It returns the run and the options that
// the choice sets, and false where the maintainer left: a choice that names no step is a
// maintainer who left.
func pick(ctx context.Context, c Call, run Run) (Run, step.Options, bool, error) {
	scan := func(ctx context.Context, b *tui.Bridge) error {
		_, err := run.Do(ctx, Effects{M: b.Effects().M, Out: b, Text: io.Discard}, b.Options(step.Options{Check: true}))
		return err
	}
	choice, err := tui.Pick(ctx, ScreenConfig(c), c.Path, run.Groups(true)[run.Listed:], scan)
	if err != nil || len(choice.Only) == 0 {
		return run, step.Options{}, false, err
	}
	chosen, o, err := run.Choose(choice)
	return chosen, o, err == nil, err
}
