// Package lazygit is the lazygit role as Go steps: lazygit from Ubuntu's archive, and the wrapper
// in front of it. See docs/lazygit.md.
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
)

// wrapperPath comes before the package's /usr/bin/lazygit on every PATH of Ubuntu's.
const wrapperPath = "/usr/local/bin/lazygit"

//go:embed files/lazygit
var wrapper []byte

// Steps returns the role's steps in order. The role takes no secrets.
func Steps(_ values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	return []step.Step{
		host.Group{Title: packageStep, Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true, Names: []string{"lazygit"}},
		}},
		host.Group{Title: wrapperStep, After: packageStep, Parts: []host.Resource{
			host.File{M: m, Path: wrapperPath, Content: wrapper, Mode: 0o755},
		}},
	}
}
