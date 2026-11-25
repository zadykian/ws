package tunnel_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/roles/tunnel"
)

func TestForNamesTheFilesOfOneServersDaemon(t *testing.T) {
	want := tunnel.Daemon{
		Name: "lab", Label: "com.github.zadykian.ws-tunnel.lab",
		Plist: "/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.lab.plist",
		Dir:   "/etc/ws/tunnel/lab", Routes: "/etc/ws/tunnel/lab/routes.d", Conf: "/etc/ws/tunnel/lab/tunnel.conf",
		State: "/var/run/ws-tunnel.lab.state", Log: "/var/log/ws-tunnel.lab.log",
	}
	if got := tunnel.For("lab"); got != want {
		t.Errorf("For(lab) = %+v, want %+v", got, want)
	}
}

func TestInstalledNamesTheServersThatHaveAPlist(t *testing.T) {
	m := effecttest.NewMachine()
	if got, err := tunnel.Installed(m); err != nil || len(got) != 0 {
		t.Errorf("Installed = %q, %v; want none where the directory is missing", got, err)
	}
	dir := "/Library/LaunchDaemons/"
	m.Dirs[dir[:len(dir)-1]] = 0o755
	for _, f := range []string{"com.github.zadykian.ws-tunnel.ws.plist", "com.github.zadykian.ws-tunnel.lab.plist",
		"com.github.zadykian.ws-tunnel.plist", "com.github.zadykian.ws-tunnel.a.b.plist", "com.apple.x.plist"} {
		m.Files[dir+f] = []byte("x")
	}
	got, err := tunnel.Installed(m)
	if err != nil || len(got) != 2 || got[0] != "lab" || got[1] != "ws" {
		t.Errorf("Installed = %q, %v; want lab and ws alone", got, err)
	}
}

func TestFoundJoinsThePlistsStatesLogsAndDirectories(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/Library/LaunchDaemons"], m.Dirs["/var/run"], m.Dirs["/var/log"] = 0o755, 0o755, 0o755
	m.Dirs[tunnel.Dir], m.Dirs[tunnel.Dir+"/box"], m.Dirs[tunnel.Dir+"/ws"] = 0o755, 0o755, 0o755
	for _, f := range []string{
		"/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist",
		"/var/run/ws-tunnel.lab.state", "/var/run/ws-tunnel.lab.state.new", "/var/run/ws-tunnel.state",
		"/var/log/ws-tunnel.old.log", "/var/log/ws-tunnel.log", "/var/log/system.log",
		tunnel.Dir + "/private.key",
	} {
		m.Files[f] = []byte("x")
	}
	got, err := tunnel.Found(m)
	if want := []string{"box", "lab", "old", "ws"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("Found = %q, %v; want %q", got, err, want)
	}
	for _, c := range m.Calls {
		if c[:4] == "root" {
			t.Errorf("Found ran %q, want no root", c)
		}
	}
}

func TestRereadSignalsEachLoadedDaemonButTheOnesSkipped(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/Library/LaunchDaemons"] = 0o755
	for _, n := range []string{"ws", "lab", "off", "old"} {
		m.Files["/Library/LaunchDaemons/"+tunnel.For(n).Label+".plist"] = []byte("x")
	}
	for _, n := range []string{"ws", "lab", "old"} {
		m.Results["launchctl print system/"+tunnel.For(n).Label] = effecttest.Result{}
	}
	cs, err := tunnel.Reread(context.Background(), m, nil, "old")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range cs {
		got = append(got, c.String())
	}
	want := []string{"send SIGHUP to system/" + tunnel.For("lab").Label, "send SIGHUP to system/" + tunnel.For("ws").Label}
	if !slices.Equal(got, want) {
		t.Errorf("changes = %q, want %q", got, want)
	}
}

func TestRereadWarnsWhereTheSignalIsRefused(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/Library/LaunchDaemons"] = 0o755
	m.Files["/Library/LaunchDaemons/"+tunnel.For("ws").Label+".plist"] = []byte("x")
	m.Results["launchctl print system/"+tunnel.For("ws").Label] = effecttest.Result{}
	out := &effecttest.Out{}
	cs, err := tunnel.Reread(context.Background(), m, out)
	if err != nil || len(cs) != 1 {
		t.Fatalf("Reread = %v, %v; want one signal", cs, err)
	}
	if err := (resource.Group{}).Apply(context.Background(), step.Plan{Changes: cs}); err != nil || len(out.Lines) != 1 {
		t.Errorf("Apply = %v, said %q; want a warning alone", err, out.Lines)
	}
}
