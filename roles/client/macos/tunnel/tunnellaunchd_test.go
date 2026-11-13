package tunnel_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/tunnel"
)

const doneConf = "# Written by github.com/zadykian/ws: ws client macos apply, from ~/.config/ws/mac.conf.\nWS_PUBLIC_KEY=K\nWS_PORT=51820\nWS_SLOT=0\n"

// daemonMachine is a Mac with ws, root's copy, the key, tunnel.conf and the plist in place.
func daemonMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/h/.local/bin/ws"] = []byte("ws")
	m.Files["/usr/local/libexec/ws-tunnel/ws"] = []byte("ws")
	m.Files["/etc/ws-tunnel/ws/tunnel.conf"] = []byte(doneConf)
	m.Files["/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist"] = []byte("plist")
	for path, mode := range map[string]effect.Attrs{
		"/usr/local": {Mode: 0o755}, "/usr/local/libexec": {Mode: 0o755}, "/usr/local/libexec/ws-tunnel": {Mode: 0o755},
		"/usr/local/libexec/ws-tunnel/ws": {Mode: 0o755}, "/etc/ws-tunnel": {Mode: 0o700}, "/etc/ws-tunnel/ws": {Mode: 0o700},
		"/etc/ws-tunnel/routes.d": {Mode: 0o755}, "/etc/ws-tunnel/private.key": {Mode: 0o600},
		"/etc/ws-tunnel/ws/tunnel.conf":                                 {Mode: 0o600},
		"/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist": {Mode: 0o644},
	} {
		m.Root[path] = mode
	}
	m.Results["/usr/local/libexec/ws-tunnel/ws client macos tunnel key"] = effecttest.Result{Out: []byte("PUB\n")}
	m.Files["/h/.config/ws/tunnel.pub"] = []byte("PUB\n")
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"] = 0o755, 0o700
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	return m
}

func daemonStep(m *effecttest.Machine) *tunnel.Tunnel {
	s := tunnelStep(m)
	s.Conf = conf.Conf{PublicKey: "K"}
	s.Plist = []byte("plist")
	s.Sleep = func(time.Duration) {}
	return s
}

func TestTunnelPlansNothingOnceTheDaemonRuns(t *testing.T) {
	if got := tunnelChanges(t, daemonStep(daemonMachine())); len(got) != 0 {
		t.Errorf("Plan = %q; want none", got)
	}
}

func TestTunnelBootstrapsTheDaemonWhereLaunchdLacksIt(t *testing.T) {
	m := daemonMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{Err: errors.New("no")}
	got := lines(tunnelChanges(t, daemonStep(m)))
	want := "would run launchctl bootstrap system /Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist"
	if got != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
}

func TestTunnelSignalsTheDaemonWhereTunnelConfAloneChanged(t *testing.T) {
	m := daemonMachine()
	m.Files["/etc/ws-tunnel/ws/tunnel.conf"] = []byte("old\n")
	got := tunnelChanges(t, daemonStep(m))
	if len(got) != 2 || got[1] != "would run launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel.ws" {
		t.Errorf("Plan = %q; want the conf and a SIGHUP", got)
	}
}

func TestTunnelBootsTheDaemonOutAndInWherePlistChanged(t *testing.T) {
	s := daemonStep(daemonMachine())
	s.Plist = []byte("new")
	got := tunnelChanges(t, s)
	if len(got) != 3 || got[1] != "would run launchctl bootout system/com.github.zadykian.ws-tunnel.ws" {
		t.Errorf("Plan = %q; want the plist, bootout and bootstrap", got)
	}
}

func TestTunnelKicksTheDaemonWhereWsChanged(t *testing.T) {
	m := daemonMachine()
	m.Files["/usr/local/libexec/ws-tunnel/ws"] = []byte("old")
	got := tunnelChanges(t, daemonStep(m))
	want := "would run launchctl kickstart -k system/com.github.zadykian.ws-tunnel.ws"
	if len(got) != 2 || got[1] != want {
		t.Errorf("Plan = %q; want the copy and a kickstart", got)
	}
}

func TestTunnelStartsTheDaemonWhereSIGHUPFails(t *testing.T) {
	m := daemonMachine()
	m.Files["/etc/ws-tunnel/ws/tunnel.conf"] = []byte("old\n")
	m.Results["launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{Err: errors.New("no")}
	m.Results["launchctl kickstart -k system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	s := daemonStep(m)
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	out := s.Out.(*effecttest.Out).Lines
	if last := out[len(out)-1]; last != "say: ran launchctl kickstart -k system/com.github.zadykian.ws-tunnel.ws" {
		t.Errorf("out = %q; want the kickstart", out)
	}
}

func TestTunnelTriesABootstrapTenTimes(t *testing.T) {
	m := daemonMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{Err: errors.New("no")}
	m.Results["launchctl bootstrap system /Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist"] =
		effecttest.Result{Err: errors.New("busy")}
	s := daemonStep(m)
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = s.Apply(context.Background(), p)
	if err == nil || len(m.Calls) < 10 {
		t.Errorf("Apply = %v after %d calls; want an error after 10 tries", err, len(m.Calls))
	}
}

func TestTunnelRendersThePlistWithTheServersName(t *testing.T) {
	s := daemonStep(effecttest.NewMachine())
	s.Server, s.Plist = "lab", tunnel.Plist
	got := lines(tunnelChanges(t, s))
	for _, want := range []string{"ws-tunnel.lab</string>", "<string>--server</string>\n+\t\t<string>lab</string>", "/var/log/ws-tunnel.lab.log"} {
		if !strings.Contains(got, want) {
			t.Errorf("Plan lacks %q", want)
		}
	}
	if strings.Contains(got, "@NAME@") {
		t.Error("Plan holds a name left unfilled")
	}
}
