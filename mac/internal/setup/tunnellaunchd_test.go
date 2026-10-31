package setup_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

const doneConf = "# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf.\nWS_PUBLIC_KEY=K\nWS_PORT=51820\n"

// daemonMachine is a Mac with ws, root's copy, the key, tunnel.conf and the plist in place.
func daemonMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/h/.local/bin/ws"] = []byte("ws")
	m.Files["/usr/local/libexec/ws-tunnel/ws"] = []byte("ws")
	m.Files["/etc/ws-tunnel/tunnel.conf"] = []byte(doneConf)
	m.Files["/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist"] = []byte("plist")
	m.Files["/m/plist"] = []byte("plist")
	for path, mode := range map[string]effect.Attrs{
		"/usr/local": {Mode: 0o755}, "/usr/local/libexec": {Mode: 0o755}, "/usr/local/libexec/ws-tunnel": {Mode: 0o755},
		"/usr/local/libexec/ws-tunnel/ws": {Mode: 0o755}, "/etc/ws-tunnel": {Mode: 0o700},
		"/etc/ws-tunnel/routes.d": {Mode: 0o755}, "/etc/ws-tunnel/private.key": {Mode: 0o600},
		"/etc/ws-tunnel/tunnel.conf":                                 {Mode: 0o600},
		"/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist": {Mode: 0o644},
	} {
		m.Root[path] = mode
	}
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{}
	return m
}

func daemonStep(m *effecttest.Machine) *setup.Tunnel {
	s := tunnelStep(m)
	s.Conf = conf.Conf{PublicKey: "K"}
	s.Plist = "/m/plist"
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
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{Err: errors.New("no")}
	got := lines(tunnelChanges(t, daemonStep(m)))
	want := "would run launchctl bootstrap system /Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist"
	if got != want {
		t.Errorf("Plan = %q; want %q", got, want)
	}
}

func TestTunnelSignalsTheDaemonWhereTunnelConfAloneChanged(t *testing.T) {
	m := daemonMachine()
	m.Files["/etc/ws-tunnel/tunnel.conf"] = []byte("old\n")
	got := tunnelChanges(t, daemonStep(m))
	if len(got) != 2 || got[1] != "would run launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel" {
		t.Errorf("Plan = %q; want the conf and a SIGHUP", got)
	}
}

func TestTunnelBootsTheDaemonOutAndInWherePlistChanged(t *testing.T) {
	m := daemonMachine()
	m.Files["/m/plist"] = []byte("new")
	got := tunnelChanges(t, daemonStep(m))
	if len(got) != 3 || got[1] != "would run launchctl bootout system/com.github.zadykian.ws-tunnel" {
		t.Errorf("Plan = %q; want the plist, bootout and bootstrap", got)
	}
}

func TestTunnelKicksTheDaemonWhereWsChanged(t *testing.T) {
	m := daemonMachine()
	m.Files["/usr/local/libexec/ws-tunnel/ws"] = []byte("old")
	got := tunnelChanges(t, daemonStep(m))
	want := "would run launchctl kickstart -k system/com.github.zadykian.ws-tunnel"
	if len(got) != 2 || got[1] != want {
		t.Errorf("Plan = %q; want the copy and a kickstart", got)
	}
}

func TestTunnelStartsTheDaemonWhereSIGHUPFails(t *testing.T) {
	m := daemonMachine()
	m.Files["/etc/ws-tunnel/tunnel.conf"] = []byte("old\n")
	m.Results["launchctl kill SIGHUP system/com.github.zadykian.ws-tunnel"] = effecttest.Result{Err: errors.New("no")}
	m.Results["launchctl kickstart -k system/com.github.zadykian.ws-tunnel"] = effecttest.Result{}
	s := daemonStep(m)
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	out := s.Out.(*effecttest.Out).Lines
	if last := out[len(out)-1]; last != "say: ran launchctl kickstart -k system/com.github.zadykian.ws-tunnel" {
		t.Errorf("out = %q; want the kickstart", out)
	}
}

func TestTunnelTriesABootstrapTenTimes(t *testing.T) {
	m := daemonMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{Err: errors.New("no")}
	m.Results["launchctl bootstrap system /Library/LaunchDaemons/com.github.zadykian.ws-tunnel.plist"] =
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
