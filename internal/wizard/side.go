package wizard

import "github.com/zadykian/ws/engine/core/step"

// Side is the machine that a stage acts on. The wizard's order crosses the two, so a screen
// that shows a list for each reads it from the stage.
type Side string

// The two machines of the wizard. A stage that changes both is the Mac's where its last change
// lands on the Mac, as in the Tunnel stage, whose last write is WS_PUBLIC_KEY in mac.conf.
const (
	OnMac Side = "Mac"
	OnWs  Side = "ws"
)

// Sided is a stage that names its machine.
type Sided interface {
	Side() Side
}

// SideOf returns the machine that a stage names, and false where it names none.
func SideOf(s step.Step) (Side, bool) {
	p, ok := s.(Sided)
	if !ok {
		return "", false
	}
	return p.Side(), true
}

// Side is the Mac: the stage installs ws's tool before the others.
func (*MacUpdate) Side() Side { return OnMac }

// Side is the Mac: the Command Line Tools.
func (*Tools) Side() Side { return OnMac }

// Side is the Mac: Homebrew.
func (*Homebrew) Side() Side { return OnMac }

// Side is the Mac: mac.conf.
func (*Answers) Side() Side { return OnMac }

// Side is the Mac: its ssh key.
func (*MacKey) Side() Side { return OnMac }

// Side is the Mac: gh's login and the Mac's key among the account's.
func (*Gh) Side() Side { return OnMac }

// Side is the Mac: ws client macos apply.
func (*Setup) Side() Side { return OnMac }

// Side is the Mac: known_hosts holds ws's host key.
func (*HostKey) Side() Side { return OnMac }

// Side is the Mac: the stage checks the Mac's login and changes nothing on ws.
func (*Login) Side() Side { return OnMac }

// Side is ws: how its scripts reach root.
func (*Sudo) Side() Side { return OnWs }

// Side is ws: this release is installed there.
func (*WsInstall) Side() Side { return OnWs }

// Side is ws: its host_vars.
func (*WsValues) Side() Side { return OnWs }

// Side is ws: ws server ubuntu apply runs there.
func (*WsRun) Side() Side { return OnWs }

// Side is ws: the claude login of WS_USER.
func (*WsClaude) Side() Side { return OnWs }

// Side is its GitHub's: the Mac, whose gh login takes ws's signing key, or ws.
func (s *Signing) Side() Side { return s.GitHub.side() }

// Side is the Mac: the stage ends in the Mac's WireGuard configuration and mac.conf.
func (*Tunnel) Side() Side { return OnMac }

// Side is the Mac: the cable's service, after ws's port in host_vars.yml.
func (*Cable) Side() Side { return OnMac }
