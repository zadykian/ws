package docker

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
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

func TestRouteCheckStandsInAReport(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[ipRoutes] = effecttest.Result{Out: []byte(`[{"dst":"172.16.5.0/24","dev":"wg0","table":"7"}]`)}
	group := host.Group{Title: routesStep, Parts: []host.Resource{routeCheck{m, values.Default().Docker}}}
	reports, err := step.Read(t.Context(), []step.Step{group})
	if err != nil || len(reports) != 1 || len(reports[0].Items) != 1 {
		t.Fatalf("reports = %+v, %v; want one item", reports, err)
	}
	if got := reports[0].Items[0]; got.Kind != step.CheckItem || !strings.Contains(got.Conflict, "172.16.5.0/24 dev wg0 table 7") {
		t.Errorf("item = %+v, want a check that names the route", got)
	}
}

func TestRouteCheckPassesClearRoutes(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[ipRoutes] = effecttest.Result{Out: []byte(`[{"dst":"default","dev":"eth0"}]`)}
	if changes, err := (routeCheck{m, values.Default().Docker}).Plan(t.Context()); err != nil || changes != nil {
		t.Errorf("Plan = %v, %v; want none", changes, err)
	}
}
