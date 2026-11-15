package networks_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
	"github.com/zadykian/ws/roles/client/macos/networks"
)

const resolverHead = "# Written by github.com/zadykian/ws: ws client macos apply, from WS_RESOLVERS in a server's server.conf.\n"

func installDaemons(m *effecttest.Machine, names ...string) {
	m.Dirs["/Library/LaunchDaemons"] = 0o755
	for _, n := range names {
		m.Files["/Library/LaunchDaemons/"+launchd.For(n).Label+".plist"] = []byte("x")
	}
}

func networksStep(m *effecttest.Machine, c conf.Conf) *networks.Networks {
	m.Root["/etc/ws-tunnel/ws/routes.d"] = effect.Attrs{Mode: 0o755}
	installDaemons(m, "ws")
	return &networks.Networks{Server: "ws", Conf: c, M: m, Out: &effecttest.Out{}}
}

func networksPlan(t *testing.T, s *networks.Networks) []string {
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
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	s := networksStep(m, conf.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24", Resolvers: "a.test=10.0.0.1"})
	got := networksPlan(t, s)
	if len(got) != 4 || !strings.Contains(got[0], "+10.1.0.0/24") ||
		got[1] != "would run launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel.ws" ||
		got[2] != "create directory /etc/resolver, owner 0:0, mode 755" || !strings.Contains(got[3], "+nameserver 10.0.0.1") {
		t.Errorf("Plan = %q", got)
	}
}

func TestNetworksSaysWhatCheckWouldPutBeforeRoutesDExists(t *testing.T) {
	m := effecttest.NewMachine()
	s := &networks.Networks{Server: "ws", Conf: conf.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24 10.2.0.0/24"}, M: m, Out: &effecttest.Out{}, Check: true}
	got := networksPlan(t, s)
	want := "would put 10.1.0.0/24 10.2.0.0/24 in /etc/ws-tunnel/ws/routes.d/server-conf once routes.d is made"
	if len(got) != 1 || got[0] != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
	s.Check = false
	if _, err := s.Plan(context.Background()); err == nil ||
		err.Error() != "/etc/ws-tunnel/ws/routes.d is missing: run ws client macos apply --server ws first" {
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

func TestNetworksKeepsTheResolverOfADomainAnotherServerNames(t *testing.T) {
	m := effecttest.NewMachine()
	for _, f := range []string{"gone.test", "lab.test"} {
		m.Files["/etc/resolver/"+f] = []byte(resolverHead + "nameserver 10.0.0.1\n")
		m.Root["/etc/resolver/"+f] = effect.Attrs{Mode: 0o644}
	}
	m.Root["/etc/resolver"] = effect.Attrs{Mode: 0o755}
	m.Dirs["/etc/resolver"] = 0o755
	const mac = "/h/.config/ws/mac.conf"
	for _, n := range []string{"ws", "lab"} {
		m.Dirs["/h/.config/ws/servers"], m.Dirs["/h/.config/ws/servers/"+n] = 0o755, 0o755
		m.Files[conf.ServerPath(mac, n)] = nil
	}
	m.Files[conf.ServerPath(mac, "lab")] = []byte("WS_RESOLVERS=lab.test=10.0.0.1\n")
	s := networksStep(m, conf.Conf{PublicKey: "K"})
	s.MacConf = mac
	got := networksPlan(t, s)
	if len(got) != 1 || !strings.Contains(got[0], "/etc/resolver/gone.test") {
		t.Errorf("Plan = %q; want gone.test's removal alone", got)
	}
}

func TestNetworksWarnsWhereTheDaemonCannotBeSignalled(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	m.Results["launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{Err: context.Canceled}
	s := networksStep(m, conf.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24"})
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	out := s.Out.(*effecttest.Out).Lines
	if last := out[len(out)-1]; !strings.HasPrefix(last, "warn: cannot signal com.github.zadykian.ws-tunnel.ws") {
		t.Errorf("out = %q; want the warning", out)
	}
}

func TestNetworksMakesResolverDirOnceForSeveralDomains(t *testing.T) {
	got := networksPlan(t, networksStep(effecttest.NewMachine(), conf.Conf{PublicKey: "K", Resolvers: "a.test=10.0.0.1 b.test=10.0.0.2"}))
	n := 0
	for _, c := range got {
		if strings.HasPrefix(c, "create directory /etc/resolver,") {
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

func TestNetworksApplySaysEachChangeInOrder(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	m.Results["launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	s := networksStep(m, conf.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24"})
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	out := s.Out.(*effecttest.Out).Lines
	if len(out) != 2 || !strings.Contains(out[0], "/etc/ws-tunnel/ws/routes.d/server-conf") ||
		out[1] != "say: ran launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel.ws" {
		t.Errorf("out = %q; want the routes file, then the signal", out)
	}
}
