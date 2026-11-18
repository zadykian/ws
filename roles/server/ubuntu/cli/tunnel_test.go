package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
)

const wg0Key = "/etc/systemd/network/60-wg0.key"

func TestTunnelCheckSaysOnStdout(t *testing.T) {
	p, _ := valuesProcess("---\n")
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "tunnel", "check")
	if status != 0 || stderr != "" || !strings.Contains(stdout, "wg0 leaves out the peer mac, as /etc/wireguard/mac.pub is missing") {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}

func TestTunnelCheckRefusesBadValues(t *testing.T) {
	p, _ := valuesProcess("---\nWS_TUNNEL_PORT: 0\n")
	if status, _, stderr := runWs(p, "", "server", "ubuntu", "tunnel", "check"); status != 1 || !strings.Contains(stderr, "WS_TUNNEL_PORT is 0") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

func TestTunnelKeyCheckDueWithoutKey(t *testing.T) {
	p, m := valuesProcess("---\n")
	m.Files["/etc/group"] = []byte("systemd-network:x:998:\n")
	status, _, stderr := runWs(p, "", "server", "ubuntu", "tunnel", "key", "--check")
	if status != 1 || !strings.Contains(stderr, "to do: make the server's private key") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
	if _, made := m.Files[wg0Key]; made {
		t.Error("--check made the key")
	}
}

func TestTunnelKeyMadeOnce(t *testing.T) {
	p, m := valuesProcess("---\n")
	m.Files["/etc/group"] = []byte("systemd-network:x:998:\n")
	if status, _, stderr := runWs(p, "", "server", "ubuntu", "tunnel", "key"); status != 0 {
		t.Fatalf("status = %d, stderr %q", status, stderr)
	}
	key := string(m.Files[wg0Key])
	if a, _ := m.RootAttrs(wg0Key); key == "" || a != (effect.Attrs{GID: 998, Mode: 0o640}) {
		t.Fatalf("attrs %+v, want root:998 0640", a)
	}
	if strings.Contains(strings.Join(m.Calls, "\n"), strings.TrimSpace(key)) {
		t.Error("a call shows the key")
	}
	for _, args := range [][]string{{"--check"}, nil} {
		status, _, stderr := runWs(p, "", append([]string{"server", "ubuntu", "tunnel", "key"}, args...)...)
		if status != 0 || stderr != "" {
			t.Errorf("%q again: status = %d, stderr %q", args, status, stderr)
		}
	}
	if string(m.Files[wg0Key]) != key {
		t.Error("the key changed")
	}
}
