package zsh

import (
	"context"
	"errors"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

// Step is zsh as a step of resources.
func (z *Zsh) Step() resource.Lazy {
	return resource.Lazy{Title: "zsh", Parts: z.parts}
}

// zshRun is what the parts of one run tell the next. Parts makes a new one each time.
type zshRun struct {
	z      *Zsh
	linked error // ~/.zshrc is a link that lacks the line
	err    error // what stops the plan, after its warning
}

// zshPart reports what its file holds, and plans by one method of the run.
type zshPart struct {
	step.Reporter
	plan func(context.Context) ([]resource.Change, error)
}

// Plan returns the part's changes.
func (p zshPart) Plan(ctx context.Context) ([]resource.Change, error) { return p.plan(ctx) }

// parts is /etc/ws, the hook, the line and the warning, in the order of a plan. It reads nothing and
// warns of nothing.
func (z *Zsh) parts(context.Context) ([]resource.Resource, error) {
	r := &zshRun{z: z}
	return append(z.hookParts(), zshPart{z.rcLines(), r.lines}, zshWarnings{r}), nil
}

// lines plans the line of ~/.zshrc. It keeps what the warning says.
func (r *zshRun) lines(ctx context.Context) ([]resource.Change, error) {
	changes, err := r.z.rcLines().Plan(ctx)
	if errors.Is(err, file.ErrLinked) {
		r.linked, err = err, nil
	}
	r.err = err
	return changes, nil
}

// zshWarnings is the last part: it warns of a link, and returns what stopped the line. It reports
// nothing, so that status shows no warning.
type zshWarnings struct{ r *zshRun }

// Plan plans no change.
func (p zshWarnings) Plan(context.Context) ([]resource.Change, error) {
	r := p.r
	if r.linked != nil {
		r.z.Out.Warn("%s", r.linked)
	}
	return nil, r.err
}
