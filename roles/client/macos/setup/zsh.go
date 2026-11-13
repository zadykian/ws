package setup

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

const (
	// zshHeader is the hook's first line, by which a run knows the file for its own.
	zshHeader = "# Written by ws client macos apply: its next run replaces changes made here."
	// zshLine is the line of ~/.zshrc that sources the hook.
	zshLine = "source ~/.config/ws/zshrc"
)

// Zsh writes ~/.config/ws/zshrc, a hook that turns off at each prompt the modes that a dropped ssh
// leaves on (docs/mac-ghostty.md). It adds the line that sources the hook at the end of ~/.zshrc,
// and keeps the old file as ~/.zshrc.TIME~, as the ssh step keeps ~/.ssh/config. It warns, and leaves
// ~/.zshrc, where the hook is not ws's or ~/.zshrc is a link that lacks the line.
type Zsh struct {
	Home string
	// Hook is roles/client/macos/mac/zshrc, its first line zshHeader.
	Hook []byte
	M    effect.Machine
	Out  effect.Out
	// Now stamps the copy of ~/.zshrc. Nil means time.Now.
	Now func() time.Time
}

// Name is zsh.
func (*Zsh) Name() string { return "zsh" }

// WaitsOn is nothing.
func (*Zsh) WaitsOn() string { return "" }

func (z *Zsh) hook() string { return z.Home + "/.config/ws/zshrc" }

func (z *Zsh) rc() string { return z.Home + "/.zshrc" }

// zshOwn says whether the hook's first line is ws's.
func zshOwn(have []byte) bool {
	first, _, _ := strings.Cut(string(have), "\n")
	return first == zshHeader
}

// zshSources says whether a line of text is zshLine, spaces aside.
func zshSources(text []byte) bool {
	for _, line := range strings.Split(string(text), "\n") {
		if strings.TrimSpace(line) == zshLine {
			return true
		}
	}
	return false
}

// Plan plans ~/.config and ~/.config/ws, as the git step does, then the hook and the line.
func (z *Zsh) Plan(ctx context.Context) (step.Plan, error) {
	cs, link, err := WsDirs(z.M, z.Home).Plan()
	if err != nil {
		return step.Plan{}, err
	}
	hook, err := z.hookFile().Plan()
	switch {
	case errors.Is(err, ErrNotWs):
		z.Out.Warn("%s is not ws's: move it away, and the next run writes ws's and sources it", z.hook())
		link.warn(z.Out, len(cs), "ws's files go there")
		return step.Plan{Changes: cs}, nil
	case err != nil:
		return step.Plan{}, err
	}
	cs = append(cs, hook...)
	line, err := z.rcLines().Plan(ctx)
	if errors.Is(err, ErrLinked) {
		z.Out.Warn("%s", err)
		err = nil
	}
	cs = append(cs, line...)
	link.warn(z.Out, len(cs), "ws's files go there")
	return step.Plan{Changes: cs}, err
}

// rcLines is zshLine at the end of ~/.zshrc.
func (z *Zsh) rcLines() Lines {
	return Lines{
		M: z.M, Path: z.rc(), Lines: zshLine + "\n", Shown: zshLine, Mode: 0o644, Now: z.Now,
		Has: func(_ context.Context, _ effect.Machine, _ string, text []byte) (bool, error) {
			return zshSources(text), nil
		},
	}
}

// Apply makes the directories and writes the files.
func (z *Zsh) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := change.Apply(ctx, z.M, z.Out, c); err != nil {
			return err
		}
	}
	return nil
}

// Report says how the hook and the line stand. A ~/.zshrc that is a link lacking the line stands
// against ws: a run only warns of it.
func (z *Zsh) Report(ctx context.Context) ([]step.Item, error) {
	return Gather(ctx, WsDirs(z.M, z.Home), z.hookFile(), z.rcLines())
}

// hookFile is the hook, mode 644.
func (z *Zsh) hookFile() File {
	return File{M: z.M, Path: z.hook(), Text: z.Hook, Mode: 0o644, Own: zshOwn}
}
