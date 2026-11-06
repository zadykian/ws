// Package fresh is the fresh role as Go steps: the editor from its release's .deb, and its config. See
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
	configStep   = "Fresh config"
)

// Steps returns the role's steps in order. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Lazy{Title: downloadStep, Parts: download(m)},
		host.Lazy{Title: installStep, After: downloadStep, Parts: install(m)},
		user.Owned(m, configStep, "", func(uid, gid int) ([]host.Resource, error) {
			return configFiles(m, v, uid, gid)
		}),
	}
}
