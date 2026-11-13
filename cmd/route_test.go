package cmd

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
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{Err: errors.New("not loaded")}
	if status, _, stderr := runWs(p, "", "client", "macos", "tunnel", "loaded"); status != 1 ||
		stderr != "ws client macos tunnel loaded: launchd has not loaded the daemon com.github.zadykian.ws-tunnel\n" {
		t.Errorf("not loaded: status = %d, stderr = %q; want 1", status, stderr)
	}
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{}
	if status, _, stderr := runWs(p, "", "client", "macos", "tunnel", "loaded"); status != 0 || stderr != "" {
		t.Errorf("loaded: status = %d, stderr = %q; want 0", status, stderr)
	}
}

func TestTunnelRoutesCheckShowsTheFileAndWritesNothing(t *testing.T) {
	p := fakeProcess()
	m := p.Machine.(*effecttest.Machine)
	m.Root["/etc/ws-tunnel/routes.d"] = effect.Attrs{Mode: 0o755}
	status, _, stderr := runWs(p, "", "client", "macos", "tunnel", "routes", "--check", "test", "198.51.100.0/24")
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
	status, _, stderr := runWs(fakeProcess(), "", "client", "macos", "tunnel", "routes", "test", "0.0.0.0/0")
	if want := "ws client macos tunnel routes: test: 0.0.0.0/0 overlaps the tunnel's 10.99.0.0/24\n"; status != 1 || stderr != want {
		t.Errorf("routes: status = %d, stderr = %q; want 1, %q", status, stderr, want)
	}
}
