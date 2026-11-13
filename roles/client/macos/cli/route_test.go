package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestTunnelLoadedFollowsLaunchd(t *testing.T) {
	p := fakeProcess()
	m := p.Machine.(*effecttest.Machine)
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{Err: errors.New("not loaded")}
	if status, _, stderr := runWs(p, "", "client", "macos", "tunnel", "loaded", "--server", "ws"); status != 1 ||
		stderr != "ws client macos tunnel loaded: launchd has not loaded the daemon com.github.zadykian.ws-tunnel.ws\n" {
		t.Errorf("not loaded: status = %d, stderr = %q; want 1", status, stderr)
	}
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	if status, _, stderr := runWs(p, "", "client", "macos", "tunnel", "loaded", "--server", "ws"); status != 0 || stderr != "" {
		t.Errorf("loaded: status = %d, stderr = %q; want 0", status, stderr)
	}
}

func TestTunnelRoutesCheckShowsTheFileAndWritesNothing(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"WS_CONF": "/e/mac.conf"})
	m := p.Machine.(*effecttest.Machine)
	m.Root["/etc/ws-tunnel/ws/routes.d"] = effect.Attrs{Mode: 0o755}
	m.Files["/e/servers/ws/server.conf"] = []byte("WS_USER=ada\n")
	withServerDirs(m)
	status, _, stderr := runWs(p, "", "client", "macos", "tunnel", "routes", "--server", "ws", "--check", "test", "198.51.100.0/24")
	if status != 0 || !strings.Contains(stderr, "+198.51.100.0/24") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "write") {
			t.Errorf("--check made %q", c)
		}
	}
}

func TestRouteCommandsPrintTheErrorAndExit1(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"WS_CONF": "/e/mac.conf"})
	m := p.Machine.(*effecttest.Machine)
	m.Files["/e/servers/ws/server.conf"] = []byte("WS_USER=ada\n")
	withServerDirs(m)
	status, _, stderr := runWs(p, "", "client", "macos", "tunnel", "routes", "--server", "ws", "test", "0.0.0.0/0")
	if want := "ws client macos tunnel routes: test: 0.0.0.0/0 overlaps the tunnels' 10.99.0.0/20\n"; status != 1 || stderr != want {
		t.Errorf("routes: status = %d, stderr = %q; want 1, %q", status, stderr, want)
	}
	status, _, stderr = runWs(p, "", "client", "macos", "tunnel", "routes", "test", "10.1.0.0/24")
	want := "ws client macos tunnel routes: --server is required: it names the server whose tunnel takes the networks\n"
	if status != 1 || stderr != want {
		t.Errorf("no --server: status = %d, stderr = %q; want 1, %q", status, stderr, want)
	}
}
