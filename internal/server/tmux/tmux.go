// Package tmux is the tmux role as Go steps: tmux from Ubuntu's archive, in place of its snap. See
// docs/tmux.md.
package tmux

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

// Steps returns the role's steps: the package, then the snap's removal, so that the machine never
// lacks a tmux. No config goes with it: cld runs tmux with -f /dev/null. Where cld's sessions run
// the snap's tmux, end them before a real run. cld then finds /usr/bin/tmux first on the PATH, and
// their key bindings call /snap/bin/tmux, which the removal takes away. The role takes no values
// and no secrets.
func Steps(_ values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	return []step.Step{
		host.Group{Title: "Tmux", Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true, Names: []string{"tmux"}},
			Snap{M: m, Out: out},
		}},
	}
}
