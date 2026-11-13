// Package macos runs the Mac's steps in order, as ws client macos apply, list and state do.
package macos

import (
	"context"
	"io"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/ghostty"
	"github.com/zadykian/ws/roles/client/macos/git"
	"github.com/zadykian/ws/roles/client/macos/mac"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

// Mac is what the Mac's steps read and act through. It holds the order of ws client macos apply's
// steps, so that no caller builds them itself.
type Mac struct {
	Conf conf.Conf
	Home string
	// XDG is XDG_CONFIG_HOME, which may be empty.
	XDG string
	M   effect.Machine
	Out effect.Out
	// Text takes what a run prints on stdout.
	Text io.Writer
}

// steps returns the steps in the order ws client macos apply runs them. Under check, the steps that
// plan again after they apply plan once.
func (m Mac) steps(check bool) []step.Step {
	return []step.Step{
		m.ssh(false),
		setup.Cable{Conf: m.Conf, M: m.M, Out: m.Out},
		&git.Git{Conf: m.Conf, Home: m.Home, XDG: m.XDG, M: m.M, Out: m.Out},
		&ghostty.Ghostty{Home: m.Home, Settings: ghostty.Settings, M: m.M, Out: m.Out},
		&setup.Zed{Home: m.Home, Settings: mac.ZedSettings, Keymap: mac.ZedKeymap, M: m.M, Out: m.Out},
		&setup.Cld{Home: m.Home, M: m.M, Out: m.Out, Check: check},
		&setup.Zsh{Home: m.Home, Hook: mac.Zshrc, M: m.M, Out: m.Out, Now: time.Now},
		&setup.Tunnel{Conf: m.Conf, Home: m.Home, Plist: mac.TunnelPlist, M: m.M, Out: m.Out, Text: m.Text,
			Check: check},
		&setup.Networks{Conf: m.Conf, M: m.M, Out: m.Out, Check: check},
	}
}

// Roles returns the steps of steps(check) as roles of one step each, named by the step, in order.
func (m Mac) Roles(check bool) []role.Role {
	var roles []role.Role
	for _, s := range m.steps(check) {
		roles = append(roles, role.Role{Name: s.Name(), Steps: func(effect.Machine, effect.Units, effect.Out) []step.Step {
			return []step.Step{s}
		}})
	}
	return roles
}

// Names returns the names of the steps, in order: the rows of a list.
func Names() []string {
	var names []string
	for _, s := range (Mac{Text: io.Discard}).steps(false) {
		names = append(names, s.Name())
	}
	return names
}

// State reads the steps' reports under a check: it applies nothing and prints nothing on stdout.
func (m Mac) State(ctx context.Context) ([]step.Report, error) {
	m.Text = io.Discard
	return step.Read(ctx, m.steps(true))
}
