package otelcol

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestStepsSayTheySkipWithoutSystemd(t *testing.T) {
	m := machine(version + " ii\n")
	out := &effecttest.Out{}
	if p, err := (answer{m: m, out: out}).Plan(t.Context()); err != nil || p != nil {
		t.Errorf("health without systemd = %v, %v; want none", p, err)
	}
	if p, err := (stopped{m: m, out: out}).Plan(t.Context()); err != nil || p != nil {
		t.Errorf("stop without systemd = %v, %v; want none", p, err)
	}
	if len(out.Lines) != 2 {
		t.Errorf("said %q; want a skip for each", out.Lines)
	}
}
