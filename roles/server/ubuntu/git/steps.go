package git

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// keyStep is the signing key's step, which the allowed signers wait on.
const keyStep = "Git signing key"

// Steps returns the role's steps in order: the key that signs commits, the allowed signers, the
// config, the tokens and the clones. Each runs as WS_USER.
func Steps(v values.Values, s secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	user := account.User{Name: v.User, Home: v.Home}
	return []step.Step{
		user.WithAccount(m, keyStep, "", func(a account.Account) ([]resource.Resource, error) {
			return SigningKey(m, a, v.GitSigningPubKey), nil
		}),
		user.WithAccount(m, "Git allowed signers", keyStep, func(a account.Account) ([]resource.Resource, error) {
			return AllowedSigners(m, a, v.GitSigningPubKey), nil
		}),
		user.WithAccount(m, "Git config", "", func(a account.Account) ([]resource.Resource, error) {
			return []resource.Resource{Config{M: m, Out: out, A: a, Key: v.GitSigningPubKey}}, nil
		}),
		user.WithAccount(m, "Git tokens", "", func(a account.Account) ([]resource.Resource, error) {
			return []resource.Resource{Tokens{M: m, A: a, Creds: Credentials(s.Git), Whole: s.Whole}}, nil
		}),
		user.WithAccount(m, "Git clones", "", func(a account.Account) ([]resource.Resource, error) {
			return []resource.Resource{Clones{M: m, A: a, Clones: v.Git.Clones}}, nil
		}),
	}
}
