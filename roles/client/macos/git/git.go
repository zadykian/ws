// Package git sets up git's identity and signing key in ~/.config/ws/gitconfig, and gh.
package git

import (
	"context"
	"os/exec"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
)

// Git sets git up from mac.conf's WS_GIT_NAME, WS_GIT_EMAIL and WS_GIT_SIGNING_PUB_KEY, and leaves it
// alone without them. It writes git's identity and a signing key of the Mac's own to
// ~/.config/ws/gitconfig, as the shell role sets them up on ws. ~/.gitconfig includes that file at
// its top. gh, from Homebrew where missing, takes git's pushes to GitHub, and registers the key
// there. It stops where the key or ~/.gitconfig is not as it can use, before it acts. Without a
// login for gh, or its write:ssh_signing_key scope, it warns, at each run until one adds the key.
type Git struct {
	Conf conf.Conf
	Home string
	// XDG is git's XDG config directory, XDG_CONFIG_HOME or ~/.config, whose git/config git reads
	// before ~/.gitconfig.
	XDG string
	M   effect.Machine
	Out effect.Out
	Now func() time.Time
	// LookPath finds a program on the user's PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)

	notes []string        // the warnings of the plan in the making
	known []string        // the lines for what the plan found there already
	said  map[string]bool // the warnings said in this run
	// addErr is why the key's add failed in this run.
	addErr error
	// added is whether this run added the key, so that the plan after it says no "already".
	added bool
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

// Plan reads the signing key and gh, and plans to install gh.
func (g *Git) Plan(ctx context.Context) (step.Plan, error) {
	if g.Conf.GitName == "" {
		return step.Plan{}, nil
	}
	g.notes = nil
	found, link, holds, err := g.gitconfigLines().Read(ctx)
	lines, err := g.gitconfigLines().Changes(found, link, holds, err)
	if err != nil {
		return step.Plan{}, err
	}
	key, err := keys.ReadKey(ctx, g.M, g.Conf, g.Home, keys.SigningKey)
	if err != nil {
		return step.Plan{}, err
	}
	gh, install, err := g.gh(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	var cs []step.Change
	if install != nil {
		cs = append(cs, *install)
	}
	files, err := g.files(key, gh)
	if err != nil {
		return step.Plan{}, err
	}
	cs = append(append(cs, files...), g.includeChanges(ctx, found, lines)...)
	reg, waits, err := g.register(ctx, key, gh, install != nil)
	if err != nil {
		return step.Plan{}, err
	}
	if reg != nil {
		cs = append(cs, *reg)
	} else if waits != "" {
		cs = append(cs, change.Remark(waits))
	}
	return step.Plan{Changes: g.withNotes(cs)}, nil
}

// Apply installs gh first, plans again with it there, then applies the rest as a group does.
func (g *Git) Apply(ctx context.Context, p step.Plan) error {
	n := 0
	for n < len(p.Changes) {
		if _, ok := p.Changes[n].(ghInstall); !ok {
			break
		}
		n++
	}
	if n > 0 {
		if err := (resource.Group{}).Apply(ctx, step.Plan{Changes: p.Changes[:n]}); err != nil {
			return err
		}
		var err error
		if p, err = g.Plan(ctx); err != nil {
			return err
		}
		n = 0
	}
	return resource.Group{}.Apply(ctx, step.Plan{Changes: p.Changes[n:]})
}
