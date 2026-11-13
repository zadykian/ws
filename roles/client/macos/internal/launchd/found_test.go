package launchd_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

func TestFoundJoinsThePlistsStatesLogsAndDirectories(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/Library/LaunchDaemons"], m.Dirs["/var/run"], m.Dirs["/var/log"] = 0o755, 0o755, 0o755
	for _, f := range []string{
		"/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist",
		"/var/run/ws-tunnel.lab.state", "/var/run/ws-tunnel.lab.state.new", "/var/run/ws-tunnel.state",
		"/var/log/ws-tunnel.old.log", "/var/log/ws-tunnel.log", "/var/log/system.log",
	} {
		m.Files[f] = []byte("x")
	}
	m.Root[launchd.Dir] = effect.Attrs{Mode: 0o700}
	m.Results["ls -1 "+launchd.Dir] = effecttest.Result{Out: []byte("box\nprivate.key\nroutes.d\ntunnel.conf\nws\n")}
	got, err := launchd.Found(context.Background(), m)
	if want := []string{"box", "lab", "old", "ws"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("Found = %q, %v; want %q", got, err, want)
	}
}
