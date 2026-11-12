package signoz

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

func TestEachStepWaitsOnTheOneBeforeItBarDashboards(t *testing.T) {
	steps := Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
	var names []string
	for i, s := range steps {
		before := ""
		if i > 0 {
			before = names[i-1]
		}
		if s.Name() == "signoz dashboards" {
			before = "signoz ready" // the retention is no precondition of the dashboards
		}
		if w, ok := s.(interface{ WaitsOn() string }); ok && i > 0 && w.WaitsOn() != before {
			t.Errorf("%s waits on %q, want %q", s.Name(), w.WaitsOn(), before)
		}
		names = append(names, s.Name())
	}
	want := []string{"signoz files", "signoz password", "signoz up", "signoz ready", "signoz retention", "signoz dashboards"}
	if !slices.Equal(names, want) {
		t.Errorf("steps = %q, want %q", names, want)
	}
}
