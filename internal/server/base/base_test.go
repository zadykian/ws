package base

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

func TestStepsAreTheRolesInOrder(t *testing.T) {
	var names []string
	for _, s := range Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{}) {
		names = append(names, s.Name())
	}
	want := []string{"Packages", "Locale", "Timezone", "Upgrades", "Swap", "Swappiness"}
	if !slices.Equal(names, want) {
		t.Errorf("steps = %v, want %v", names, want)
	}
}

func TestDefaultsAreTheRolesValues(t *testing.T) {
	b := values.Default().Base
	if b.Locale != "en_US.UTF-8" || b.Timezone != "Etc/UTC" || b.Swappiness != 10 || b.SwapSizeMB != nil {
		t.Errorf("defaults = %+v", b)
	}
}
