package wizard

import "github.com/zadykian/ws/engine/core/step"

// Side is the machine that a stage acts on. The wizard's order crosses the two, so a screen
// that shows a list for each reads it from the stage.
type Side string

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
