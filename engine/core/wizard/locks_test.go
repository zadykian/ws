package wizard

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

// waiting is a stage that waits on first.
type waiting struct{ step.Step }

func (waiting) WaitsOn() string { return "first" }

// locker is a stage that locks another than the one it waits on.
type locker struct{ waiting }

func (locker) Locks() string { return "second" }

// Picking a stage locks the one it waits on, or the one that its Locks names.
func TestLocksNamesTheLockedStage(t *testing.T) {
	if got := Locks(waiting{}); got != "first" {
		t.Errorf("a waiting stage locks %q, want first", got)
	}
	if got := Locks(locker{}); got != "second" {
		t.Errorf("a locking stage locks %q, want second", got)
	}
}
