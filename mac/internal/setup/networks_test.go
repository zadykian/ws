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

const resolverHead = "# Written by ws's mac/setup.sh from WS_RESOLVERS in ~/.config/ws/mac.conf.\n"

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
	if got := networksPlan(t, networksStep(effecttest.NewMachine(), conf.Conf{PublicKey: "K"})); len(got) != 0 {
		t.Errorf("Plan = %q; want none", got)
	}
}

func TestNetworksWritesRoutesAndResolversAndSignalsTheDaemon(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{}
	s := networksStep(m, conf.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24", Resolvers: "a.test=10.0.0.1"})
	got := networksPlan(t, s)
	if len(got) != 4 || !strings.Contains(got[0], "+10.1.0.0/24") ||
		got[1] != "would run launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel" ||
		got[2] != "would make /etc/resolver, root's, with mode 755" || !strings.Contains(got[3], "+nameserver 10.0.0.1") {
		t.Errorf("Plan = %q", got)
	}
}

func TestNetworksSaysWhatCheckWouldPutBeforeRoutesDExists(t *testing.T) {
	m := effecttest.NewMachine()
	s := &setup.Networks{Conf: conf.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24 10.2.0.0/24"}, M: m, Out: &effecttest.Out{}, Check: true}
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

func TestNetworksStopsAtAResolverOfAnotherScript(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/resolver/a.test"] = []byte("nameserver 1.1.1.1\n")
	m.Root["/etc/resolver/a.test"] = effect.Attrs{Mode: 0o644}
	_, err := networksStep(m, conf.Conf{PublicKey: "K", Resolvers: "a.test=10.0.0.1"}).Plan(context.Background())
	if err == nil || !strings.HasPrefix(err.Error(), "/etc/resolver/a.test is another script's, or written by hand") {
		t.Errorf("Plan error = %v", err)
	}
}

func TestNetworksRemovesTheResolverOfADomainMacConfDropped(t *testing.T) {
	m := effecttest.NewMachine()
	for _, f := range []string{"gone.test", "kept.test", "theirs.test"} {
		m.Files["/etc/resolver/"+f] = []byte(resolverHead + "nameserver 10.0.0.1\n")
		m.Root["/etc/resolver/"+f] = effect.Attrs{Mode: 0o644}
	}
	m.Root["/etc/resolver"] = effect.Attrs{Mode: 0o755}
	m.Dirs["/etc/resolver"] = 0o755
	m.Files["/etc/resolver/theirs.test"] = []byte("nameserver 1.1.1.1\n")
	got := networksPlan(t, networksStep(m, conf.Conf{PublicKey: "K", Resolvers: "kept.test=10.0.0.1"}))
	if len(got) != 1 || !strings.Contains(got[0], "/etc/resolver/gone.test") {
		t.Errorf("Plan = %q; want gone.test's removal alone", got)
	}
}

func TestNetworksWarnsWhereTheDaemonCannotBeSignalled(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{}
	m.Results["launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel"] = effecttest.Result{Err: context.Canceled}
	s := networksStep(m, conf.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24"})
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

func TestNetworksMakesResolverDirOnceForSeveralDomains(t *testing.T) {
	got := networksPlan(t, networksStep(effecttest.NewMachine(), conf.Conf{PublicKey: "K", Resolvers: "a.test=10.0.0.1 b.test=10.0.0.2"}))
	n := 0
	for _, c := range got {
		if strings.HasPrefix(c, "would make /etc/resolver,") {
			n++
		}
	}
	if n != 1 || len(got) != 3 {
		t.Errorf("Plan = %q; want one mkdir and two files", got)
	}
}

func TestNetworksPlansNothingUntilMacConfHasWsKey(t *testing.T) {
	m := effecttest.NewMachine()
	s := networksStep(m, conf.Conf{TunnelNetworks: "10.1.0.0/24", Resolvers: "a.test=10.0.0.1"})
	if got := networksPlan(t, s); len(got) != 0 {
		t.Errorf("Plan = %q; want none", got)
	}
}
