package brew

import (
	"context"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// App is the cask of an app as the part that the parts after it wait on. Plan keeps whether the
// app is there or comes, and Then gives the parts that need it.
type App struct {
	Cask
	comes bool
}

// Plan plans the cask, as Cask does, and keeps whether the app is there or comes.
func (a *App) Plan(ctx context.Context) ([]resource.Change, error) {
	changes, comes, err := a.plan(ctx)
	a.comes = comes
	return changes, err
}

// Then gives r as a part that plans where the app is there or comes, else nothing. A report shows
// all. It never wraps a change, so that a failed install of the app still reaches the group, which
// holds the changes after it and lets the font's go on.
func (a *App) Then(r resource.Resource) resource.Resource { return waiting{a, r} }

type waiting struct {
	app *App
	r   resource.Resource
}

func (w waiting) Plan(ctx context.Context) ([]resource.Change, error) {
	if !w.app.comes {
		return nil, nil
	}
	return w.r.Plan(ctx)
}

// Report is the part's own, whether the app comes or not.
func (w waiting) Report(ctx context.Context) ([]step.Item, error) {
	if rep, ok := w.r.(step.Reporter); ok {
		return rep.Report(ctx)
	}
	return nil, nil
}
