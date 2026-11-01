package daemon

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/tunnel"
)

func TestRunRoutes(t *testing.T) {
	a, b, c := netip.MustParsePrefix("10.1.0.0/16"), netip.MustParsePrefix("10.2.0.0/16"), netip.MustParsePrefix("10.3.0.0/16")
	m := effecttest.NewMachine()
	m.Results["route -q -n delete -inet 10.1.0.0/16"] = effecttest.Result{Err: errors.New("not in table")}
	m.Results["route -q -n delete -inet 10.2.0.0/16"] = effecttest.Result{}
	m.Results["route -q -n add -inet 10.3.0.0/16 -interface utun4"] = effecttest.Result{Err: errors.New("exists")}
	m.Results["route -q -n add -inet 10.4.0.0/16 -interface utun4"] = effecttest.Result{}
	d := netip.MustParsePrefix("10.4.0.0/16")
	failed, logs := RunRoutes(context.Background(), m, tunnel.RoutePlan{Delete: []netip.Prefix{a, b}, Add: []netip.Prefix{c, d}}, "utun4")
	if !slices.Equal(failed, []netip.Prefix{c}) {
		t.Errorf("failed = %v, want [%v]", failed, c)
	}
	if !slices.Equal(logs, []string{"cannot remove the route to 10.1.0.0/16"}) {
		t.Errorf("logs = %q", logs)
	}
	want := []string{
		"run route -q -n delete -inet 10.1.0.0/16", "run route -q -n delete -inet 10.2.0.0/16",
		"run route -q -n add -inet 10.3.0.0/16 -interface utun4", "run route -q -n add -inet 10.4.0.0/16 -interface utun4",
	}
	if !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}
