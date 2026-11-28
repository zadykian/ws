package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/tunnel"
)

func TestTunnelYieldNeedsRoot(t *testing.T) {
	p := fakeProcess()
	if status, _, stderr := runWs(p, "macos", "tunnel", "yield", "10.20.0.0/16"); status != 1 ||
		stderr != "ws macos tunnel yield: run it as root, with sudo\n" {
		t.Errorf("status = %d, stderr = %q; want the refusal", status, stderr)
	}
}

func TestTunnelYieldRefusesABadNetworkAndWritesNothing(t *testing.T) {
	for _, bad := range []string{"10.20.0.1/16", "10.20.0.0", "fd00::/8", "x"} {
		p := fakeProcess()
		p.Euid = 0
		m := p.Machine.(*effecttest.Machine)
		status, _, stderr := runWs(p, "macos", "tunnel", "yield", "10.20.0.0/16", bad)
		if status != 1 || !strings.Contains(stderr, "'"+bad+"' is not an IPv4 network, as 192.0.2.0/24") {
			t.Errorf("%s: status = %d, stderr = %q; want the refusal", bad, status, stderr)
		}
		if _, ok := m.Files[tunnel.YieldFile]; ok || len(m.Calls) > 0 {
			t.Errorf("%s: calls = %q, want none", bad, m.Calls)
		}
	}
}

func TestTunnelYieldWritesTheFileAndSignals(t *testing.T) {
	p := fakeProcess()
	p.Euid = 0
	m := machineWithDaemons("ws")
	p.Machine = m
	label := tunnel.For("ws").Label
	m.Results["launchctl print system/"+label] = effecttest.Result{}
	m.Results["launchctl kill SIGHUP system/"+label] = effecttest.Result{}
	m.Files[tunnel.For("ws").State] = []byte("tier=lan\nyielded=10.20.0.0/16\n")
	if status, stdout, stderr := runWs(p, "macos", "tunnel", "yield", "10.20.0.0/16"); status != 0 || stdout != "" || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0 and no line", status, stdout, stderr)
	}
	if got := string(m.Files[tunnel.YieldFile]); got != "10.20.0.0/16\n" {
		t.Errorf("the yield = %q", got)
	}
	m.Results["launchctl kill SIGHUP system/"+label] = effecttest.Result{Err: errors.New("exit status 3")}
	if status, _, stderr := runWs(p, "macos", "tunnel", "yield"); status != 1 || !strings.Contains(stderr, "cannot signal system/"+label) {
		t.Errorf("a failed signal: status = %d, stderr = %q; want it named", status, stderr)
	}
}
