package devtools

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestStepsOrderAndWaits(t *testing.T) {
	steps := Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
	var names, waits []string
	for _, s := range steps {
		names, waits = append(names, s.Name()), append(waits, s.WaitsOn())
	}
	if want := []string{"Devtools gh telemetry", "Devtools gh login", "Devtools roslyn-language-server", "Devtools npm servers", "Devtools rustup"}; !slices.Equal(names, want) {
		t.Errorf("steps %q, want %q", names, want)
	}
	if want := []string{"", "", "", "", ""}; !slices.Equal(waits, want) {
		t.Errorf("waits %q, want %q", waits, want)
	}
}

func TestEachStepWaitsOnAnEarlierOne(t *testing.T) {
	steps := Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
	for i, s := range steps {
		if w := s.WaitsOn(); w != "" && !slices.ContainsFunc(steps[:i], func(e step.Step) bool { return e.Name() == w }) {
			t.Errorf("%q waits on %q, which is not an earlier step", s.Name(), w)
		}
	}
}
