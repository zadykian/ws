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
	z     *Zsh
	link  file.UserDirLink
	n     int  // the changes planned so far
	notWs bool // the hook is not ws's, so the line waits
}

// zshPart reports what its files hold, and plans by one method of the run.
type zshPart struct {
	step.Reporter
	plan func(context.Context) ([]resource.Change, error)
}

// Plan returns the part's changes.
func (p zshPart) Plan(ctx context.Context) ([]resource.Change, error) { return p.plan(ctx) }

// parts is the directories, the hook and the line, in the order of a plan. It reads nothing.
func (z *Zsh) parts(context.Context) ([]resource.Resource, error) {
	r := &zshRun{z: z}
	return []resource.Resource{zshPart{file.WsDirs(z.M, z.Out, z.Home), r.dirs},
		zshPart{z.hookFile(), r.hook}, zshPart{z.rcLines(), r.lines}}, nil
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

// lines plans the line of ~/.zshrc, and warns of the hook, then of the links, where it found them.
func (r *zshRun) lines(ctx context.Context) ([]resource.Change, error) {
	z := r.z
	if r.notWs {
		z.Out.Warn("%s is not ws's: move it away, and the next run writes ws's and sources it", z.hook())
		r.link.Warn(z.Out, r.n, "ws's files go there")
		return nil, nil
	}
	changes, err := z.rcLines().Plan(ctx)
	if errors.Is(err, file.ErrLinked) {
		z.Out.Warn("%s", err)
		err = nil
	}
	r.link.Warn(z.Out, r.n+len(changes), "ws's files go there")
	return changes, err
}
