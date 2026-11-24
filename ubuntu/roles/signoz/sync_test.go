package signoz

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

func TestSyncRunsReadyFirst(t *testing.T) {
	var names, waits []string
	for _, s := range Sync(values.Default().Signoz, effecttest.NewMachine(), &effecttest.Out{}) {
		names, waits = append(names, s.Name()), append(waits, s.WaitsOn())
	}
	if want := []string{"signoz ready", "signoz retention", "signoz dashboards"}; !slices.Equal(names, want) {
		t.Errorf("steps = %q, want %q", names, want)
	}
	// The retention is no precondition of the dashboards: both wait on the API alone.
	if want := []string{"", "signoz ready", "signoz ready"}; !slices.Equal(waits, want) {
		t.Errorf("waits = %q, want %q", waits, want)
	}
}
