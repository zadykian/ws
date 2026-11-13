package tunnel_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/tunnel"
)

// laterMachine is a Mac with the daemon of ws, a server's file for ws, and the leftovers of the
// server lab: its plist, directory, state file and log, loaded.
func laterMachine() *effecttest.Machine {
	m := daemonMachine()
	m.Files["/h/.config/ws/servers/ws/server.conf"] = []byte("x")
	m.Dirs["/h/.config/ws/servers"], m.Dirs["/h/.config/ws/servers/ws"] = 0o755, 0o755
	otherDaemon(m, "lab")
	m.Dirs["/var/run"], m.Dirs["/var/log"] = 0o755, 0o755
	for _, f := range []string{"/var/run/ws-tunnel.lab.state", "/var/log/ws-tunnel.lab.log", "/etc/ws-tunnel/lab/tunnel.conf"} {
		m.Files[f] = []byte("x")
		m.Root[f] = effect.Attrs{Mode: 0o600}
	}
	m.Root["/etc/ws-tunnel/lab"] = effect.Attrs{Mode: 0o700}
	m.Root["/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.lab.plist"] = effect.Attrs{Mode: 0o644}
	m.Results["ls -1 /etc/ws-tunnel"] = effecttest.Result{Out: []byte("lab\nprivate.key\nws\n")}
	return m
}

func pruneStep(m *effecttest.Machine) *tunnel.Tunnel {
	s := daemonStep(m)
	s.MacConf = "/h/.config/ws/mac.conf"
	return s
}

func TestTunnelRemovesTheDaemonOfAServerThatIsGone(t *testing.T) {
	got := lines(tunnelChanges(t, pruneStep(laterMachine())))
	for _, want := range []string{
		"would run launchctl bootout system/com.github.zadykian.ws-tunnel.lab",
		"remove /Library/LaunchDaemons/com.github.zadykian.ws-tunnel.lab.plist", "remove /etc/ws-tunnel/lab",
		"remove /var/run/ws-tunnel.lab.state", "remove /var/log/ws-tunnel.lab.log",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Plan lacks %q:\n%s", want, got)
		}
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "remove ") || strings.Contains(line, "bootout") {
			if strings.Contains(line, "/ws-tunnel/ws") || strings.Contains(line, "ws-tunnel.ws") {
				t.Errorf("Plan removes the daemon of ws: %q", line)
			}
		}
	}
}

func TestTunnelLeavesTheDaemonsAloneWhereTheServersLackItsOwn(t *testing.T) {
	m := laterMachine()
	delete(m.Files, "/h/.config/ws/servers/ws/server.conf")
	if got := tunnelChanges(t, pruneStep(m)); len(got) != 0 {
		t.Errorf("Plan = %q; want none where servers/ does not list ws", got)
	}
}

func TestTunnelPrunesNothingTheSecondTimeAndAppliesNothingInPlan(t *testing.T) {
	m := laterMachine()
	m.Results["launchctl bootout system/com.github.zadykian.ws-tunnel.lab"] = effecttest.Result{}
	s := pruneStep(m)
	tunnelChanges(t, s)
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root bootout") || strings.HasPrefix(c, "root rm") {
			t.Fatalf("Plan ran %q", c)
		}
	}
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	m.Results["ls -1 /etc/ws-tunnel"] = effecttest.Result{Out: []byte("private.key\nws\n")}
	if got := tunnelChanges(t, pruneStep(m)); len(got) != 0 {
		t.Errorf("second Plan = %q; want none", got)
	}
}

func TestTunnelDoesNotKickADaemonItPrunes(t *testing.T) {
	m := laterMachine()
	m.Files["/usr/local/libexec/ws-tunnel/ws"] = []byte("old")
	for _, c := range tunnelChanges(t, pruneStep(m)) {
		if strings.Contains(c, "kickstart") && strings.Contains(c, "lab") {
			t.Errorf("Plan kicks lab: %q", c)
		}
	}
}

func TestTunnelHasTheDaemonsThatStayReadAgainOnceAServerGoes(t *testing.T) {
	m := laterMachine()
	m.Files["/h/.config/ws/servers/box/server.conf"] = []byte("x")
	m.Dirs["/h/.config/ws/servers/box"] = 0o755
	otherDaemon(m, "box")
	m.Results["ls -1 /etc/ws-tunnel"] = effecttest.Result{Out: []byte("box\nlab\nprivate.key\nws\n")}
	got := lines(tunnelChanges(t, pruneStep(m)))
	if !strings.Contains(got, "would run launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel.box") {
		t.Errorf("Plan does not signal box:\n%s", got)
	}
	if strings.Contains(got, "kill SIGHUP system/com.github.zadykian.ws-tunnel.lab") {
		t.Errorf("Plan signals the daemon it removes:\n%s", got)
	}
}
