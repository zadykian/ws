// Package fresh is the fresh role as Go steps: the editor from its release's .deb, the wrapper that
// carries its keys through tmux, and its config and theme. See docs/fresh.md.
package fresh

import (
	"embed"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Step names. Those after the download wait on the steps before them.
const (
	downloadStep = "Fresh download"
	installStep  = "Fresh install"
	wrapperStep  = "Fresh wrapper"
	configStep   = "Fresh config"
)

// wrapperPath comes before the .deb's /usr/bin/fresh on every PATH of Ubuntu's, as docs/fresh.md says.
const wrapperPath = "/usr/local/bin/fresh"

// templates holds config.json.tmpl, which the role fills with the user's home.
//
//go:embed files/config.json.tmpl
var templates embed.FS

//go:embed files/fresh
var wrapper []byte

// themeFile is JetBrains Dark as a Fresh theme. A theme is strict JSON: it takes no header.
//
//go:embed files/themes/jetbrains-dark.json
var themeFile []byte

// Steps returns the role's steps in order. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Lazy{Title: downloadStep, Parts: download(m)},
		host.Group{Title: installStep, After: downloadStep, Parts: []host.Resource{pinned(m)}},
		host.Group{Title: wrapperStep, After: installStep, Parts: []host.Resource{
			host.File{M: m, Path: wrapperPath, Content: wrapper, Mode: 0o755},
		}},
		user.WithAccount(m, configStep, "", func(a host.Account) ([]host.Resource, error) {
			return configFiles(m, v, a)
		}),
	}
}
