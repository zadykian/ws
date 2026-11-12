// Package shell is the shell role as Go steps: ws_user's bash, git, the key that signs commits,
// ~/.secrets/env, git's tokens and the clones. See docs/shell.md.
package shell

import (
	"runtime"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Steps returns the role's steps in order. The check for links comes first, so that no write runs
// beside one. Each step that reads its paths waits on it: a report goes past a step that fails, and
// would read through the link. Packages follow, as ble.sh and bash-completion serve the files
// after.
func Steps(v values.Values, s secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	steps := []step.Step{
		host.Group{Title: linksStep, Parts: []host.Resource{
			host.Plain{M: m, Home: v.Home, Paths: rootPaths(v)},
		}},
		host.Group{Title: "Shell packages", Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true, Names: []string{"bash-completion", "ble.sh"}},
		}},
		owned(user, m, "Shell files", linksStep, func(a account) ([]host.Resource, error) {
			return dotfiles(m, a, runtime.GOARCH)
		}),
	}
	if v.Shell.GitSigning {
		steps = append(steps, owned(user, m, keyStep, linksStep, func(a account) ([]host.Resource, error) {
			return signingKey(m, a, v.GitSigningKey), nil
		}), owned(user, m, "Shell allowed signers", keyStep, func(a account) ([]host.Resource, error) {
			return allowedSigners(m, a, v.GitSigningKey, v.Shell.GitAllowedSigners, v.Shell.GitEmail), nil
		}))
	}
	return append(steps, owned(user, m, "Shell git config", linksStep, func(a account) ([]host.Resource, error) {
		return []host.Resource{gitconfig{
			M: m, Out: out, A: a, Name: v.Shell.GitName, Email: v.Shell.GitEmail, Signing: v.Shell.GitSigning,
			Key: v.GitSigningKey, Signers: v.Shell.GitAllowedSigners,
		}}, nil
	}), owned(user, m, "Shell secret env", linksStep, func(a account) ([]host.Resource, error) {
		return []host.Resource{secretEnv{M: m, A: a, Env: s.Env, Whole: s.Whole}}, nil
	}), owned(user, m, "Shell git tokens", linksStep, func(a account) ([]host.Resource, error) {
		return []host.Resource{tokens{M: m, A: a, Creds: credentials(s.Git), Whole: s.Whole}}, nil
	}), owned(user, m, "Shell clones", "", func(a account) ([]host.Resource, error) {
		return []host.Resource{clones{M: m, A: a, Clones: v.Shell.GitClones}}, nil
	}))
}

const (
	linksStep = "Shell links"
	keyStep   = "Shell signing key"
)

// rootPaths are the files under the user's home that root reads or writes, which a link there
// would send elsewhere (docs/security.md#links-under-the-home).
func rootPaths(v values.Values) []string {
	return []string{
		v.Home + "/.bashrc", v.Home + "/.profile", v.Home + "/.gitconfig",
		v.GitSigningKey, v.GitSigningKey + ".pub", v.Shell.GitAllowedSigners,
		v.Home + "/.config/git/tokens.gitconfig", v.Home + "/.config/git/tokens", v.Home + "/.secrets/env",
		v.Home + "/.config/git/local",
	}
}
