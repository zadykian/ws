package setup

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// MakeKey makes git's signing key, with no passphrase, so that git signs unattended, in a rebase too.
type MakeKey struct{ Path string }

func (k MakeKey) String() string { return "would make git's signing key, " + k.Path }

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

// Plan reads the signing key and gh, and plans to make the key and install gh.
func (g *Git) Plan(ctx context.Context) (step.Plan, error) {
	if g.Conf.GitName == "" {
		return step.Plan{}, nil
	}
	key, err := g.key(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	gh, install, err := g.gh(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	var cs []step.Change
	if key.Make {
		cs = append(cs, MakeKey{strings.TrimSuffix(key.Pub, ".pub")})
	}
	if install != nil {
		cs = append(cs, *install)
	}
	files, err := g.files(key, gh)
	return step.Plan{Changes: append(cs, files...)}, err
}

// Apply makes the key and installs gh first, plans again with them there, then applies the rest.
func (g *Git) Apply(ctx context.Context, p step.Plan) error {
	n := 0
	for ; n < len(p.Changes); n++ {
		switch c := p.Changes[n].(type) {
		case MakeKey:
			if _, err := g.M.Run(ctx, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", c.Path); err != nil {
				return fmt.Errorf("ssh-keygen cannot make %s: %w", c.Path, err)
			}
			g.Out.Say("made git's signing key, %s", c.Path)
			continue
		case Run:
			g.Out.Say("installing gh")
			if err := Apply(ctx, g.M, g.Out, c); err != nil {
				return err
			}
			continue
		}
		break
	}
	if n > 0 {
		var err error
		if p, err = g.Plan(ctx); err != nil {
			return err
		}
		n = 0
	}
	for _, c := range p.Changes[n:] {
		if _, wait := c.(Pending); wait {
			continue
		}
		if err := Apply(ctx, g.M, g.Out, c); err != nil {
			return err
		}
	}
	return nil
}
