package launchd_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

func TestLaunchctlSaysItsKind(t *testing.T) {
	if got := step.KindOf(launchd.Launchctl{}); got != step.Restart {
		t.Errorf("kind = %s, want %s", got, step.Restart)
	}
}

func launchctl(m *effecttest.Machine, out *effecttest.Out, args ...string) launchd.Launchctl {
	return launchd.Launchctl{M: m, Out: out, Sleep: func(time.Duration) {}, Args: args}
}

func TestLaunchctlSaysWhatItRan(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["launchctl kickstart -k system/x"] = effecttest.Result{}
	if err := launchctl(m, out, "kickstart", "-k", "system/x").Apply(context.Background()); err != nil ||
		len(out.Lines) != 1 || out.Lines[0] != "say: ran launchctl kickstart -k system/x" {
		t.Errorf("Apply = %v, said %q; want what it ran", err, out.Lines)
	}
}

func TestLaunchctlTriesABootstrapTenTimes(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	tries := 0
	l := launchctl(m, out, "bootstrap", "system", "p")
	l.Sleep = func(time.Duration) { tries++ }
	err := l.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "launchctl bootstrap system p failed") || tries != 9 {
		t.Errorf("Apply = %v after %d sleeps; want the failure after 9", err, tries)
	}
}

func TestLaunchctlFallsBackWhereASignalIsRefused(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["launchctl kill SIGHUP system/x"] = effecttest.Result{Err: errors.New("not running")}
	m.Results["launchctl kickstart -k system/x"] = effecttest.Result{}
	l := launchctl(m, out, "kill", "SIGHUP", "system/x")
	l.Signal, l.Label = true, "x"
	if err := l.Apply(context.Background()); err != nil || len(m.Calls) != 2 {
		t.Errorf("Apply = %v, calls %q; want a kickstart after the signal", err, m.Calls)
	}
}

func TestLaunchctlWarnsWhereTheRoutesSignalFails(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	l := launchctl(m, out, "kill", "SIGHUP", "system/x")
	l.Routes, l.Label = true, "x"
	if err := l.Apply(context.Background()); err != nil || len(out.Lines) != 1 || !strings.HasPrefix(out.Lines[0], "warn: cannot signal x,") {
		t.Errorf("Apply = %v, said %q; want a warning alone", err, out.Lines)
	}
}

func TestForNamesTheFilesOfOneServersDaemon(t *testing.T) {
	want := launchd.Daemon{
		Name: "lab", Label: "com.github.zadykian.ws-tunnel.lab",
		Plist: "/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.lab.plist",
		Dir:   "/etc/ws-tunnel/lab", Conf: "/etc/ws-tunnel/lab/tunnel.conf",
		State: "/var/run/ws-tunnel.lab.state", Log: "/var/log/ws-tunnel.lab.log",
	}
	if got := launchd.For("lab"); got != want {
		t.Errorf("For(lab) = %+v, want %+v", got, want)
	}
}

func TestInstalledNamesTheServersThatHaveAPlist(t *testing.T) {
	m := effecttest.NewMachine()
	if got, err := launchd.Installed(m); err != nil || len(got) != 0 {
		t.Errorf("Installed = %q, %v; want none where the directory is missing", got, err)
	}
	dir := "/Library/LaunchDaemons/"
	m.Dirs[dir[:len(dir)-1]] = 0o755
	for _, f := range []string{"com.github.zadykian.ws-tunnel.ws.plist", "com.github.zadykian.ws-tunnel.lab.plist",
		"com.github.zadykian.ws-tunnel.plist", "com.github.zadykian.ws-tunnel.a.b.plist", "com.apple.x.plist"} {
		m.Files[dir+f] = []byte("x")
	}
	got, err := launchd.Installed(m)
	if err != nil || len(got) != 2 || got[0] != "lab" || got[1] != "ws" {
		t.Errorf("Installed = %q, %v; want lab and ws alone", got, err)
	}
}
