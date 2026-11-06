package wizard

import "github.com/zadykian/ws/step"

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

