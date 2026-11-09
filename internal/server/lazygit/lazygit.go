// Package lazygit is the lazygit role as Go steps: lazygit from Ubuntu's archive. See
// docs/lazygit.md.
package lazygit

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Step names.
const packageStep = "Lazygit"

// Steps returns the role's steps in order. The role takes no secrets.
func Steps(_ values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	return []step.Step{
		host.Group{Title: packageStep, Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true, Names: []string{"lazygit"}},
		}},
	}
}
