package git

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestStepsNamesInOrder(t *testing.T) {
	var got []string
	for _, st := range Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{}) {
		got = append(got, st.Name())
	}
	if want := []string{"Git signing key", "Git allowed signers", "Git config", "Git tokens", "Git clones"}; !slices.Equal(got, want) {
		t.Errorf("steps = %v, want %v", got, want)
	}
}
