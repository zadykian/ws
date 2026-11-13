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
	link   file.UserDirLink
	n      int   // the changes planned so far
	notWs  bool  // the hook is not ws's, so the line waits
	linked error // ~/.zshrc is a link that lacks the line
	err    error // what stops the plan, after its warnings
}

// zshPart reports what its files hold, and plans by one method of the run.
type zshPart struct {
	step.Reporter
	plan func(context.Context) ([]resource.Change, error)
}

// Plan returns the part's changes.
func (p zshPart) Plan(ctx context.Context) ([]resource.Change, error) { return p.plan(ctx) }

// parts is the directories, the hook, the line and the warnings, in the order of a plan. It reads
// nothing and warns of nothing.
func (z *Zsh) parts(context.Context) ([]resource.Resource, error) {
	r := &zshRun{z: z}
	return []resource.Resource{zshPart{file.WsDirs(z.M, z.Out, z.Home), r.dirs},
		zshPart{z.hookFile(), r.hook}, zshPart{z.rcLines(), r.lines}, zshWarnings{r}}, nil
}

// dirs plans ~/.config and ~/.config/ws, as the git step does, and keeps the link of one.
func (r *zshRun) dirs(ctx context.Context) ([]resource.Change, error) {
	cs, link, err := file.WsDirs(r.z.M, r.z.Out, r.z.Home).Plan()
	if err != nil {
		return nil, err
	}
	r.link, r.n = link, r.n+len(cs)
	return resource.Planned{Changes: cs}.Plan(ctx)
}

// hook plans the hook. One that is not ws's plans nothing and stops the line.
func (r *zshRun) hook(ctx context.Context) ([]resource.Change, error) {
	changes, err := r.z.hookFile().Plan(ctx)
	if errors.Is(err, file.ErrNotWs) {
		r.notWs = true
		return nil, nil
	}
	r.n += len(changes)
	return changes, err
}

// lines plans the line of ~/.zshrc, where the hook is ws's. It keeps what the warnings say.
func (r *zshRun) lines(ctx context.Context) ([]resource.Change, error) {
	if r.notWs {
		return nil, nil
	}
	changes, err := r.z.rcLines().Plan(ctx)
	r.n += len(changes)
	if errors.Is(err, file.ErrLinked) {
		r.linked, err = err, nil
	}
	r.err = err
	return changes, nil
}

// zshWarnings is the last part: it warns of the hook, then of the links, and returns what stopped
// the line. It reports nothing, so that status shows no warning.
type zshWarnings struct{ r *zshRun }

// Plan plans no change.
func (p zshWarnings) Plan(context.Context) ([]resource.Change, error) {
	r, out := p.r, p.r.z.Out
	if r.notWs {
		out.Warn("%s is not ws's: move it away, and the next run writes ws's and sources it", r.z.hook())
	}
	if r.linked != nil {
		out.Warn("%s", r.linked)
	}
	r.link.Warn(out, r.n, "ws's files go there")
	return nil, r.err
}
