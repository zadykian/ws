package setup

import (
	"context"
	"os/exec"
	"time"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// Git sets git up from mac.conf's WS_GIT_NAME, WS_GIT_EMAIL and WS_GIT_SIGNING_KEY, and leaves it
// alone without them: git's identity and a signing key of the Mac's own, in ~/.config/ws/gitconfig,
// which ~/.gitconfig includes at its top, as the shell role sets them up on ws. gh, from Homebrew
// where missing, takes git's pushes to GitHub, and registers the key there. It stops where the key
// or ~/.gitconfig is not as it can use, before it acts. Without a login for gh, or its
// write:ssh_signing_key scope, it warns, at each run until one adds the key.
type Git struct {
	Conf conf.Conf
	Home string
	M    effect.Machine
	Out  effect.Out
	Now  func() time.Time
	// LookPath finds a program on the user's PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
}

// Name is git.
func (*Git) Name() string { return "git" }

// WaitsOn is nothing.
func (*Git) WaitsOn() string { return "" }

func (g *Git) look(name string) (string, error) {
	if g.LookPath != nil {
		return g.LookPath(name)
	}
	return exec.LookPath(name)
}

// Plan reads gh, and plans to install it.
func (g *Git) Plan(ctx context.Context) (step.Plan, error) {
	if g.Conf.GitName == "" {
		return step.Plan{}, nil
	}
	_, install, err := g.gh(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	var cs []step.Change
	if install != nil {
		cs = append(cs, *install)
	}
	return step.Plan{Changes: cs}, nil
}

// Apply installs gh.
func (g *Git) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if run, ok := c.(Run); ok {
			g.Out.Say("installing gh")
			if err := Apply(ctx, g.M, g.Out, run); err != nil {
				return err
			}
		}
	}
	return nil
}
