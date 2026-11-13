package wizard

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

// sided is a stage that states a side of its own.
type sided struct{ step.Step }

func (sided) Side() Side { return "far" }

func TestSideOfAStepThatStatesOne(t *testing.T) {
	if got, ok := SideOf(sided{}); !ok || got != "far" {
		t.Errorf("SideOf = %q, %v; want far", got, ok)
	}
}

func TestSideOfAStepWithNone(t *testing.T) {
	if m, ok := SideOf(nil); ok || m != "" {
		t.Errorf("SideOf(nil) = %q, %v", m, ok)
	}
}
