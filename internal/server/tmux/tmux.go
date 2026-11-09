// Package tmux is the tmux role as Go steps: tmux from Ubuntu's archive, in place of its snap. See
// docs/tmux.md.
package tmux

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Steps returns the role's steps: the package, then the snap's removal, so that the machine never
// lacks a tmux. No config goes with it: cld runs tmux with -f /dev/null. Where cld's sessions run
// the snap's tmux, end them before a real run. cld then finds /usr/bin/tmux first on the PATH, and
// their key bindings call /snap/bin/tmux, which the removal takes away. Then ws-bar, the host's
// load for the status lines of Zed's terminals and of ssh's logins. The role takes no values and
// no secrets.
func Steps(_ values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	return steps(m, effect.OSUnits{}, out, running)
}

// steps is Steps with the units and the path of ws given, for a test to fake.
func steps(m effect.Machine, u effect.Units, out effect.Out, exe func() (string, error)) []step.Step {
	return []step.Step{
		host.Group{Title: "Tmux", Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true, Names: []string{"tmux"}},
			Snap{M: m, Out: out},
		}},
		host.Lazy{Title: "Tmux's bar", Parts: barParts(m, u, out, exe)},
	}
}
