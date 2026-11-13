package cmd

import (
	"testing"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
)

func TestEffectsOfActsOnTheEffectsMachine(t *testing.T) {
	p, _ := macTerminal()
	m := effecttest.NewMachine()
	if got := effectsOf(p, engine.WizardEffects{Machine: m, Prog: "ws"}); got.p.Machine != m || got.prog != "ws" {
		t.Errorf("machine = %v, prog = %q", got.p.Machine, got.prog)
	}
}

func TestWsRunSideFindsTheRunAndNothingElse(t *testing.T) {
	find := wsRunSide(fakeProcess())
	if _, ok := find([]step.Step{stepOf{name: "one"}}); ok {
		t.Error("found a run where none is")
	}
	if run, ok := find([]step.Step{stepOf{name: "one"}, &stages.WsRun{Await: "x"}}); !ok || run.Await != "x" {
		t.Errorf("run = %+v, ok = %v", run, ok)
	}
}
