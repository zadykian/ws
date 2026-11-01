package shell

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

func stepNames(v values.Values, s secrets.Secrets) []string {
	var names []string
	for _, st := range Steps(v, s, effecttest.NewMachine(), &effecttest.Out{}) {
		names = append(names, st.Name())
	}
	return names
}

func TestStepsAreTheRolesInOrder(t *testing.T) {
	got := stepNames(values.Default(), secrets.Secrets{})
	if want := []string{"Shell packages", "Shell files"}; !slices.Equal(got, want) {
		t.Errorf("steps = %v, want %v", got, want)
	}
}
