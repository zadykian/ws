package setup_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

func networksStep(m *effecttest.Machine, c conf.Conf) *setup.Networks {
	m.Root["/etc/ws-tunnel/routes.d"] = effect.Attrs{Mode: 0o755}
	return &setup.Networks{Conf: c, M: m, Out: &effecttest.Out{}}
}

func networksPlan(t *testing.T, s *setup.Networks) []string {
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, c.String())
	}
	return got
}

func TestNetworksPlansNothingWithNoNetworksAndNoResolvers(t *testing.T) {
	if got := networksPlan(t, networksStep(effecttest.NewMachine(), conf.Conf{})); len(got) != 0 {
		t.Errorf("Plan = %q; want none", got)
	}
}

func TestNetworksWritesRoutesAndSignalsTheDaemon(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{}
	got := networksPlan(t, networksStep(m, conf.Conf{TunnelNetworks: "10.1.0.0/24"}))
	if len(got) != 2 || !strings.Contains(got[0], "+10.1.0.0/24") ||
		got[1] != "would run launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel" {
		t.Errorf("Plan = %q", got)
	}
}

func TestNetworksSaysWhatCheckWouldPutBeforeRoutesDExists(t *testing.T) {
	m := effecttest.NewMachine()
	s := &setup.Networks{Conf: conf.Conf{TunnelNetworks: "10.1.0.0/24 10.2.0.0/24"}, M: m, Out: &effecttest.Out{}, Check: true}
	got := networksPlan(t, s)
	want := "would put 10.1.0.0/24 10.2.0.0/24 in /etc/ws-tunnel/routes.d/mac-conf once routes.d is made"
	if len(got) != 1 || got[0] != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
	s.Check = false
	if _, err := s.Plan(context.Background()); err == nil ||
		err.Error() != "/etc/ws-tunnel/routes.d is missing: run ws's mac/setup.sh first" {
		t.Errorf("Plan error = %v", err)
	}
}

func TestNetworksWarnsWhereTheDaemonCannotBeSignalled(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{}
	m.Results["launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel"] = effecttest.Result{Err: context.Canceled}
	s := networksStep(m, conf.Conf{TunnelNetworks: "10.1.0.0/24"})
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	out := s.Out.(*effecttest.Out).Lines
	if last := out[len(out)-1]; !strings.HasPrefix(last, "warn: cannot signal com.github.zadykian.ws-tunnel") {
		t.Errorf("out = %q; want the warning", out)
	}
}
