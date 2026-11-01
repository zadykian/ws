package tmux

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

func TestStepsAreTheRole(t *testing.T) {
	steps := Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Out{})
	if len(steps) != 1 || steps[0].Name() != "Tmux" {
		t.Errorf("got %d steps, want Tmux alone", len(steps))
	}
}
