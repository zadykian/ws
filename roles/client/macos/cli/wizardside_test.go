package cli

import (
	"testing"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestEffectsOfActsOnTheEffectsMachine(t *testing.T) {
	p, _ := macTerminal()
	m := effecttest.NewMachine()
	if got := effectsOf(p, engine.WizardEffects{Machine: m, Prog: "ws"}); got.p.Machine != m || got.prog != "ws" {
		t.Errorf("machine = %v, prog = %q", got.p.Machine, got.prog)
	}
}
