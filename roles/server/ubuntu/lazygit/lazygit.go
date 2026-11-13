// Package lazygit is the lazygit role as Go steps: lazygit from Ubuntu's archive, the wrapper in
// front of it, and its config. See docs/lazygit.md.
package lazygit

import (
	_ "embed"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
	"github.com/zadykian/ws/engine/core/host/apt"
	"github.com/zadykian/ws/engine/core/host/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Step names. The wrapper waits on the package, as it runs the package's binary.
const (
	packageStep = "Lazygit"
	wrapperStep = "Lazygit wrapper"
	configStep  = "Lazygit config"
)

// wrapperPath comes before the package's /usr/bin/lazygit on every PATH of Ubuntu's.
const wrapperPath = "/usr/local/bin/lazygit"

//go:embed files/lazygit
var wrapper []byte

//go:embed files/config.yml
var config []byte

// Steps returns the role's steps in order. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	user := account.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Group{Title: packageStep, Parts: []host.Resource{
			apt.Packages{M: m, Out: out, Recommends: true, Names: []string{"lazygit", "git-delta"}},
		}},
		host.Group{Title: wrapperStep, After: packageStep, Parts: []host.Resource{
			file.File{M: m, Path: wrapperPath, Content: wrapper, Mode: 0o755},
		}},
		user.WithAccount(m, configStep, "", func(a account.Account) ([]host.Resource, error) {
			return configFiles(m, a), nil
		}),
	}
}

// configFiles returns ~/.config/lazygit/config.yml, which ws replaces whole, keeping the old file
// beside it. The directories on the way stay the user's.
func configFiles(m effect.Machine, a account.Account) []host.Resource {
	f := a.File(m, ".config/lazygit/config.yml", config, 0o644)
	f.Backup = true
	return []host.Resource{f}
}
