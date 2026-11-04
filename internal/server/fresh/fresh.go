// Package fresh is the fresh role as Go steps: the editor from its release's .deb. See
// docs/fresh.md.
package fresh

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Step names. The install waits on the download.
const (
	downloadStep = "Fresh download"
	installStep  = "Fresh install"
)

// Steps returns the role's steps in order. The role takes no secrets.
func Steps(_ values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Out) []step.Step {
	return []step.Step{
		host.Lazy{Title: downloadStep, Parts: download(m)},
		host.Lazy{Title: installStep, After: downloadStep, Parts: install(m)},
	}
}
