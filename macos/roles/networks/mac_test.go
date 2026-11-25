package networks_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/macos/facts"
	"github.com/zadykian/ws/macos/roles/networks"
	"github.com/zadykian/ws/macos/values"
)

// resolverHead is the header of a resolver that ws wrote.
const resolverHead = "# " + file.Mark + ": ws macos networks, from WS_RESOLVERS.\n"

// macPlan returns the headlines of the networks of servers.
func macPlan(t *testing.T, m *effecttest.Machine, servers ...facts.Server) []string {
	cs, err := networks.Mac{Servers: servers, M: m, Out: &effecttest.Out{}}.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range cs {
		got = append(got, c.String())
	}
	return got
}

func TestMacRemovesAResolverNoServerNames(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/resolver"] = 0o755
	for _, domain := range []string{"gone.test", "kept.test", "theirs.test"} {
		m.Files["/etc/resolver/"+domain] = []byte(resolverHead + "nameserver 10.0.0.1\n")
		m.Root["/etc/resolver/"+domain] = effect.Attrs{Mode: 0o644}
	}
	m.Files["/etc/resolver/theirs.test"] = []byte("nameserver 10.0.0.1\n")
	// lab has no key yet: its domain stays, as its file names it, and it writes none.
	got := macPlan(t, m, facts.Server{Name: "lab", Conf: values.Conf{Resolvers: "kept.test=10.0.0.1"}})
	if len(got) != 1 || !strings.Contains(got[0], "/etc/resolver/gone.test") {
		t.Errorf("Plan = %q; want gone.test's removal alone", got)
	}
}

func TestMacWritesEachKeyedServer(t *testing.T) {
	m := effecttest.NewMachine()
	m.Root["/etc/ws/tunnel/ws/routes.d"] = effect.Attrs{Mode: 0o755}
	got := macPlan(t, m,
		facts.Server{Name: "lab", Conf: values.Conf{TunnelNetworks: "10.2.0.0/24"}},
		facts.Server{Name: "ws", Conf: values.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24", Resolvers: "a.test=10.0.0.1"}})
	if len(got) != 3 || !strings.Contains(got[0], "+10.1.0.0/24") ||
		got[1] != "create directory /etc/resolver, owner 0:0, mode 755" || !strings.Contains(got[2], "+nameserver 10.0.0.1") {
		t.Errorf("Plan = %q; want ws's routes and resolver, and nothing of lab's", got)
	}
}

func TestMacDuesAMissingRoutesD(t *testing.T) {
	got := macPlan(t, effecttest.NewMachine(), facts.Server{Name: "ws", Conf: values.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24"}})
	if want := "would put 10.1.0.0/24 in /etc/ws/tunnel/ws/routes.d/values once routes.d is made"; len(got) != 1 || got[0] != want {
		t.Errorf("Plan = %q; want %q, as the tunnel's unit makes routes.d first", got, want)
	}
}

func TestMacStopsAtAResolverOfAnotherScript(t *testing.T) {
	for _, text := range []string{
		"nameserver 1.1.1.1\n",
		"# Written by github.com/zadykian/ws: another script, from WS_RESOLVERS in a server's values.yml.\nnameserver 1.1.1.1\n",
	} {
		m := effecttest.NewMachine()
		m.Files["/etc/resolver/a.test"] = []byte(text)
		m.Root["/etc/resolver/a.test"] = effect.Attrs{Mode: 0o644}
		_, err := networks.Mac{Servers: []facts.Server{{Name: "ws", Conf: values.Conf{PublicKey: "K", Resolvers: "a.test=10.0.0.1"}}},
			M: m, Out: &effecttest.Out{}}.Plan(context.Background())
		if err == nil || !strings.HasPrefix(err.Error(), "/etc/resolver/a.test is another script's, or written by hand") {
			t.Errorf("Plan error = %v, want the file refused", err)
		}
	}
}
