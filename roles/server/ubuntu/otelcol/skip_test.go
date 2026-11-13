package otelcol

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestStepsSayTheySkipWithoutSystemd(t *testing.T) {
	m := machine(effecttest.Package{Version: version, Status: "ii"})
	out := &effecttest.Out{}
	if p, err := (answer{m: m, out: out}).Plan(t.Context()); err != nil || p != nil {
		t.Errorf("health without systemd = %v, %v; want none", p, err)
	}
	fresh := machine(effecttest.Package{})
	p, err := (stopped{pkg: pinned(fresh), m: fresh, out: out}).Plan(t.Context())
	if err != nil || len(p) != 3 || strings.Contains(p[2].String(), "stop") {
		t.Errorf("stop without systemd = %v, %v; want the install alone", p, err)
	}
	if len(out.Lines) != 2 {
		t.Errorf("said %q; want a skip for each", out.Lines)
	}
}
