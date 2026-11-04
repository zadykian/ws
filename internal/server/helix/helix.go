// Package helix is the helix role as Go steps: hx from Ubuntu's archive, its config, which names
// the language servers of devtools. See docs/helix.md.
package helix

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

const packagesStep = "Helix packages"

// Steps returns the role's steps in order. Ubuntu's package is hx, not helix.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Group{Title: packagesStep, Parts: []host.Resource{
			host.Packages{M: m, Out: out, Names: []string{"hx"}},
		}},
		user.Owned(m, "Helix config", "", func(uid, gid int) ([]host.Resource, error) {
			return configFiles(m, v, uid, gid)
		}),
	}
}
