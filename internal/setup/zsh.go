package setup

import (
	"context"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// zshHeader is the hook's first line, by which a run knows the file for its own.
const zshHeader = "# Written by ws mac apply: its next run replaces changes made here."

// Zsh writes ~/.config/ws/zshrc, a hook that turns off at each prompt the modes that a dropped ssh
// leaves on (docs/mac-ghostty.md). It warns, and leaves the file, where the hook is not ws's.
type Zsh struct {
	Home string
	// Hook is mac/zshrc, its first line zshHeader.
	Hook []byte
	M    effect.Machine
	Out  effect.Out
}

// Name is zsh.
func (*Zsh) Name() string { return "zsh" }

// WaitsOn is nothing.
func (*Zsh) WaitsOn() string { return "" }

func (z *Zsh) hook() string { return z.Home + "/.config/ws/zshrc" }

// zshOwn says whether the hook's first line is ws's.
func zshOwn(have []byte) bool {
	first, _, _ := strings.Cut(string(have), "\n")
	return first == zshHeader
}

// Plan plans ~/.config and ~/.config/ws where missing, as the git step makes them, then the hook.
func (z *Zsh) Plan(context.Context) (step.Plan, error) {
	var cs []step.Change
	for _, d := range []struct {
		path string
		mode fs.FileMode
	}{{z.Home + "/.config", 0o755}, {z.Home + "/.config/ws", 0o700}} {
		if _, err := z.M.Stat(d.path); err != nil {
			cs = append(cs, MakeDir{d.path, d.mode})
		}
	}
	return step.Plan{Changes: append(cs, owned(z.M, z.Out, z.hook(), z.Hook, zshOwn)...)}, nil
}

// Apply makes the directories and writes the file.
func (z *Zsh) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := Apply(ctx, z.M, z.Out, c); err != nil {
			return err
		}
	}
	return nil
}

// Report says how the hook stands.
func (z *Zsh) Report(ctx context.Context) ([]step.Item, error) {
	return userFile{M: z.M, Path: z.hook(), Text: z.Hook, Mode: 0o644, Own: zshOwn}.Report(ctx)
}
