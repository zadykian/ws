// Package shell is the shell role as Go steps: ws_user's bash, git, the key that signs commits,
// ~/.secrets/env, git's tokens and the clones. See docs/shell.md.
package shell

import (
	"runtime"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Steps returns the role's steps in order. Packages come first, as ble.sh and bash-completion
// serve the files after.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	steps := []step.Step{
		host.Group{Title: "Shell packages", Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true, Names: []string{"bash-completion", "ble.sh"}},
		}},
		owned(user, m, "Shell files", "", func(a account) ([]host.Resource, error) {
			return dotfiles(m, a, runtime.GOARCH)
		}),
	}
	if v.Shell.GitSigning {
		steps = append(steps, owned(user, m, keyStep, "", func(a account) ([]host.Resource, error) {
			return signingKey(m, a, v.GitSigningKey), nil
		}), owned(user, m, "Shell allowed signers", keyStep, func(a account) ([]host.Resource, error) {
			return allowedSigners(m, a, v.GitSigningKey, v.Shell.GitAllowedSigners, v.Shell.GitEmail), nil
		}))
	}
	return steps
}

const keyStep = "Shell signing key"
