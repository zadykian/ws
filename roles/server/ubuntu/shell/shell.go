// Package shell is the shell role as Go steps: ws_user's bash, git, the key that signs commits,
// ~/.secrets/env, git's tokens and the clones. See docs/shell.md.
package shell

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
	"github.com/zadykian/ws/engine/core/host/apt"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/shell/internal/git"
	"github.com/zadykian/ws/roles/server/ubuntu/shell/internal/secretenv"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Steps returns the role's steps in order. Packages come first, as ble.sh and bash-completion serve
// the files after. Each file refuses a link on its own path, so that a link stops its step alone.
func Steps(v values.Values, s secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	user := account.User{Name: v.User, Home: v.Home}
	steps := []step.Step{
		host.Group{Title: "Shell packages", Parts: []host.Resource{
			apt.Packages{M: m, Out: out, Recommends: true, Names: []string{"bash-completion", "ble.sh"}},
		}},
		host.Lazy{Title: "Shell files", Parts: func(ctx context.Context) ([]host.Resource, error) {
			arch, err := host.Architecture(ctx, m)
			if err != nil {
				return nil, err
			}
			return user.WithAccount(m, "Shell files", "", func(a account.Account) ([]host.Resource, error) {
				return dotfiles(m, a, arch)
			}).Parts(ctx)
		}},
	}
	if v.Shell.GitSigning {
		steps = append(steps, user.WithAccount(m, keyStep, "", func(a account.Account) ([]host.Resource, error) {
			return git.SigningKey(m, a, v.GitSigningKey), nil
		}), user.WithAccount(m, "Shell allowed signers", keyStep, func(a account.Account) ([]host.Resource, error) {
			return git.AllowedSigners(m, a, v.GitSigningKey, v.Shell.GitAllowedSigners, v.Shell.GitEmail), nil
		}))
	}
	return append(steps, user.WithAccount(m, "Shell git config", "", func(a account.Account) ([]host.Resource, error) {
		return []host.Resource{git.Config{
			M: m, Out: out, A: a, Name: v.Shell.GitName, Email: v.Shell.GitEmail, Signing: v.Shell.GitSigning,
			Key: v.GitSigningKey, Signers: v.Shell.GitAllowedSigners,
		}}, nil
	}), user.WithAccount(m, "Shell secret env", "", func(a account.Account) ([]host.Resource, error) {
		return []host.Resource{secretenv.Env{M: m, A: a, Env: s.Env, Whole: s.Whole}}, nil
	}), user.WithAccount(m, "Shell git tokens", "", func(a account.Account) ([]host.Resource, error) {
		return []host.Resource{git.Tokens{M: m, A: a, Creds: git.Credentials(s.Git), Whole: s.Whole}}, nil
	}), user.WithAccount(m, "Shell clones", "", func(a account.Account) ([]host.Resource, error) {
		return []host.Resource{clones{M: m, A: a, Clones: v.Shell.GitClones}}, nil
	}))
}

const keyStep = "Shell signing key"
