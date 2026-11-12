// Package lazygit is the lazygit role as Go steps: lazygit from Ubuntu's archive, the wrapper in
// front of it, and its config. See docs/lazygit.md.
package lazygit

import (
	_ "embed"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
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
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Group{Title: packageStep, Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true, Names: []string{"lazygit", "git-delta"}},
		}},
		host.Group{Title: wrapperStep, After: packageStep, Parts: []host.Resource{
			host.File{M: m, Path: wrapperPath, Content: wrapper, Mode: 0o755},
		}},
		user.WithAccount(m, configStep, "", func(a host.Account) ([]host.Resource, error) {
			return configFiles(m, v, a.UID, a.GID), nil
		}),
	}
}

// configFiles returns ~/.config/lazygit/config.yml, which ws replaces whole, keeping the old file
// beside it. The directory stays the user's.
func configFiles(m effect.Machine, v values.Values, uid, gid int) []host.Resource {
	dir := v.Home + "/.config/lazygit"
	return []host.Resource{
		host.Plain{M: m, Home: v.Home, Paths: []string{dir + "/config.yml"}},
		host.Dir{M: m, Path: v.Home + "/.config", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
		host.Dir{M: m, Path: dir, Mode: 0o755, UID: uid, GID: gid},
		host.File{M: m, Path: dir + "/config.yml", Content: config, Mode: 0o644, UID: uid, GID: gid, Backup: true},
	}
}
