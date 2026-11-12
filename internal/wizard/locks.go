package wizard

import "github.com/zadykian/ws/engine/core/step"

// locking is a stage that, picked in the wizard's lists, locks another stage than the one it waits
// on in its run.
type locking interface {
	Locks() string
}

// Locks returns the stage that picking s in the wizard's lists locks: its Locks where it has one,
// and its WaitsOn past that.
func Locks(s step.Step) string {
	if l, ok := s.(locking); ok {
		return l.Locks()
	}
	return s.WaitsOn()
}

// Locks is ws mac apply, whose steps the stage applies itself, so a list holds them on beside it.
// A plain run's Cable reads Homebrew as ever.
func (*Cable) Locks() string { return setupStage }
