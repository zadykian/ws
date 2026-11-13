package tuitest

import (
	"syscall"

	"github.com/zadykian/ws/engine/core/step"
)

// cutError is the error of a test's Cut.
type cutError struct{ signal syscall.Signal }

// Error names the signal.
func (c cutError) Error() string { return c.signal.String() }

// Cut is a test's Cut: it returns the error of the signal.
func Cut(s syscall.Signal) error { return cutError{s} }

// OneChange is a plan's change, for the tests that read how many there are.
type OneChange string

// String is the change's text.
func (c OneChange) String() string { return string(c) }

// Planned is how a scan ends a step: with changes, under a change of the step.
func Planned(parent, name string, index int, changes ...step.Change) step.Event {
	return step.Event{Kind: step.Done, Parent: parent, Step: name, Index: index, Check: true, Changes: changes}
}
