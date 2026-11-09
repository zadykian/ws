package lazygit

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

func steps() []step.Step {
	return Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Out{})
}

func TestStepsAreTheRole(t *testing.T) {
	var names []string
	for _, s := range steps() {
		names = append(names, s.Name())
	}
	if want := []string{packageStep, wrapperStep}; !slices.Equal(names, want) {
		t.Errorf("steps %v, want %v", names, want)
	}
}

func TestTheWrapperWaitsOnThePackage(t *testing.T) {
	if got := steps()[1].WaitsOn(); got != packageStep {
		t.Errorf("the wrapper waits on %q", got)
	}
}
