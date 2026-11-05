package step_test

import (
	"testing"

	"github.com/zadykian/ws/step"
)

type removal struct{ change }

func (removal) Kind() step.ChangeKind { return step.Removal }

func TestKindOf(t *testing.T) {
	if got := step.KindOf(removal{"remove /a"}); got != step.Removal {
		t.Errorf("kind = %q, want the change's own", got)
	}
	if got := step.KindOf(change("a")); got != step.OtherChange {
		t.Errorf("kind = %q, want other for a change that says none", got)
	}
}
