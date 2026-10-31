package setup_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

func TestRoutesPutsTheNetworksInTheFileOfTheirSet(t *testing.T) {
	m := effecttest.NewMachine()
	r := &setup.Routes{Set: "lab", Networks: []string{"10.1.0.0/24"}, M: m}
	if _, err := r.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "routes.d is missing") {
		t.Errorf("Plan error = %v", err)
	}
	m.Root["/etc/ws-tunnel/routes.d"] = effect.Attrs{Mode: 0o755}
	p, err := r.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 || !strings.Contains(p.Changes[0].String(), "routes.d/lab") {
		t.Errorf("Plan = %v, %v", p, err)
	}
}

func TestRoutesRefusesWhatTheShellRefused(t *testing.T) {
	for set, networks := range map[string]string{"Lab": "10.1.0.0/24", "lab": "10.1.0.5/24", "a_b": "10.99.0.0/16",
		"x": "10.77.0.0/29", "y": "0.0.0.0/0", "z": "10.1.0.0"} {
		r := &setup.Routes{Set: set, Networks: []string{networks}, M: effecttest.NewMachine()}
		if _, err := r.Plan(context.Background()); err == nil {
			t.Errorf("Plan of %q %q succeeded", set, networks)
		}
	}
}
