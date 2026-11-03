package docker

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
)

const ipRoutes = "ip -j -4 route show table all"

func TestRouteCheckStopsOnAnOverlap(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[ipRoutes] = effecttest.Result{Out: []byte(`[{"dst":"172.16.5.0/24","dev":"wg0","table":"7"}]`)}
	_, err := routeCheck{m, values.Default().Docker}.Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "172.16.5.0/24 dev wg0 table 7") {
		t.Errorf("Plan = %v, want an error that names the route", err)
	}
}

func TestRouteCheckPassesClearRoutes(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[ipRoutes] = effecttest.Result{Out: []byte(`[{"dst":"default","dev":"eth0"}]`)}
	if changes, err := (routeCheck{m, values.Default().Docker}).Plan(t.Context()); err != nil || changes != nil {
		t.Errorf("Plan = %v, %v; want none", changes, err)
	}
}
