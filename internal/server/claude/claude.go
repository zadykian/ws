// Package claude is the claude role as Go steps: claude and cld, cld's completion in bash,
// claude's user settings and telemetry, its plugin marketplaces, and cld's restore at boot. See
// docs/claude.md.
package claude

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

const installStep = "Claude install"

// Steps returns the role's steps in order. Each of cld's steps comes after the install, as cld is
// not there to ask until a run has installed it.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Group{Title: installStep, Parts: []host.Resource{Install{M: m, User: user}}},
	}
}
