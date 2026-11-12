package setup

import (
	"io"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/step"
)

// Mac is what the Mac's steps read and act through. It holds the order of ws mac apply's steps, so
// that no caller builds them itself.
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

// Steps returns the steps in the order ws mac apply runs them. Under check, the steps that plan
// again after they apply plan once.
func (m Mac) Steps(check bool) []step.Step {
	return []step.Step{
		SSH{Conf: m.Conf, Home: m.Home, M: m.M, Out: m.Out, Now: time.Now},
		Cable{Conf: m.Conf, M: m.M, Out: m.Out},
		&Git{Conf: m.Conf, Home: m.Home, XDG: m.XDG, M: m.M, Out: m.Out},
		&Ghostty{Home: m.Home, Settings: mac.Ghostty, M: m.M, Out: m.Out},
		&Zed{Home: m.Home, Settings: mac.ZedSettings, Keymap: mac.ZedKeymap, M: m.M, Out: m.Out},
		&Cld{Home: m.Home, M: m.M, Out: m.Out, Check: check},
		&Zsh{Home: m.Home, Hook: mac.Zshrc, M: m.M, Out: m.Out, Now: time.Now},
		&Tunnel{Conf: m.Conf, Home: m.Home, Plist: mac.TunnelPlist, M: m.M, Out: m.Out, Text: m.Text,
			Check: check},
		&Networks{Conf: m.Conf, M: m.M, Out: m.Out, Check: check},
	}
}

// Names returns the names of the steps, in order: the rows of a list.
func Names() []string {
	var names []string
	for _, s := range (Mac{Text: io.Discard}).Steps(false) {
		names = append(names, s.Name())
	}
	return names
}
