// Package helix is the helix role as Go steps: hx from Ubuntu's archive, its config, which names
// the language servers of devtools, and the grammars. See docs/helix.md.
package helix

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

const packagesStep = "Helix packages"

// Steps returns the role's steps in order. Ubuntu's package is hx, not helix. git and g++ fetch and
// build the grammars.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Group{Title: packagesStep, Parts: []host.Resource{
			host.Packages{M: m, Out: out, Names: []string{"hx", "git", "g++"}},
		}},
		user.Owned(m, "Helix config", "", func(uid, gid int) ([]host.Resource, error) {
			return configFiles(m, v, uid, gid)
		}),
		user.Owned(m, "Helix grammars", "", func(uid, gid int) ([]host.Resource, error) {
			return []host.Resource{Grammars{M: m, User: user, Names: v.Helix.Grammars, UID: uid, GID: gid}}, nil
		}),
	}
}
