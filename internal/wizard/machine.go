package wizard

import "github.com/zadykian/ws/engine/core/step"

// Machine is the machine that a stage acts on. The wizard's order crosses the two, so a screen
// that shows a list for each reads it from the stage.
type Machine string

// The two machines of the wizard. A stage that changes both is the Mac's where its last change
// lands on the Mac, as in the Tunnel stage, whose last write is WS_PUBLIC_KEY in mac.conf.
const (
	OnMac Machine = "Mac"
	OnWs  Machine = "ws"
)

// Placed is a stage that names its machine.
type Placed interface {
	Machine() Machine
}

// MachineOf returns the machine that a stage names, and false where it names none.
func MachineOf(s step.Step) (Machine, bool) {
	p, ok := s.(Placed)
	if !ok {
		return "", false
	}
	return p.Machine(), true
}

// Machine is the Mac: the stage installs ws's tool before the others.
func (*MacUpdate) Machine() Machine { return OnMac }

// Machine is the Mac: the Command Line Tools.
func (*Tools) Machine() Machine { return OnMac }

// Machine is the Mac: Homebrew.
func (*Homebrew) Machine() Machine { return OnMac }

// Machine is the Mac: mac.conf.
func (*Answers) Machine() Machine { return OnMac }

// Machine is the Mac: its ssh key.
func (*MacKey) Machine() Machine { return OnMac }

// Machine is the Mac: gh's login and the Mac's key among the account's.
func (*Gh) Machine() Machine { return OnMac }

// Machine is the Mac: ws client macos apply.
func (*Setup) Machine() Machine { return OnMac }

// Machine is the Mac: known_hosts holds ws's host key.
func (*HostKey) Machine() Machine { return OnMac }

// Machine is the Mac: the stage checks the Mac's login and changes nothing on ws.
func (*Login) Machine() Machine { return OnMac }

// Machine is ws: how its scripts reach root.
func (*Sudo) Machine() Machine { return OnWs }

// Machine is ws: this release is installed there.
func (*WsInstall) Machine() Machine { return OnWs }

// Machine is ws: its host_vars.
func (*WsValues) Machine() Machine { return OnWs }

// Machine is ws: ws server ubuntu apply runs there.
func (*WsRun) Machine() Machine { return OnWs }

// Machine is ws: the claude login of WS_USER.
func (*WsClaude) Machine() Machine { return OnWs }

// Machine is its GitHub's: the Mac, whose gh login takes ws's signing key, or ws.
func (s *Signing) Machine() Machine { return s.GitHub.machine() }

// Machine is the Mac: the stage ends in the Mac's WireGuard configuration and mac.conf.
func (*Tunnel) Machine() Machine { return OnMac }

// Machine is the Mac: the cable's service, after ws's port in host_vars.yml.
func (*Cable) Machine() Machine { return OnMac }
