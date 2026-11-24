package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/wgkey"
)

const wg0Key = "/etc/systemd/network/60-wg0.key"

func TestTunnelCheckSaysOnStdout(t *testing.T) {
	p, _ := valuesProcess("---\n")
	// The check asks the units whether networkd runs: a fake answers, never the host's systemd.
	u := &effecttest.Units{States: map[string]effect.UnitState{"systemd-networkd.service": {Active: true}}}
	root := enginecli.Group("server", "the server's commands", newTunnelCmd(p, u))
	status, stdout, stderr := clitest.Run(p, []*cobra.Command{root}, "", "server", "tunnel", "check")
	if status != 0 || stderr != "" || !strings.Contains(stdout, "wg0 leaves out the peer mac, as /etc/wireguard/mac.pub is missing") {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}

func TestTunnelCheckRefusesBadValues(t *testing.T) {
	p, _ := valuesProcess("---\nWS_TUNNEL_PORT: 0\n")
	if status, _, stderr := runWs(p, "", "server", "tunnel", "check"); status != 1 || !strings.Contains(stderr, "WS_TUNNEL_PORT is 0") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

func TestTunnelKeyCheckDueWithoutKey(t *testing.T) {
	p, m := valuesProcess("---\n")
	m.Files["/etc/group"] = []byte("systemd-network:x:998:\n")
	status, _, stderr := runWs(p, "", "server", "tunnel", "key", "--check")
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
	if status, _, stderr := runWs(p, "", "server", "tunnel", "key"); status != 0 {
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
		status, _, stderr := runWs(p, "", append([]string{"server", "tunnel", "key"}, args...)...)
		if status != 0 || stderr != "" {
			t.Errorf("%q again: status = %d, stderr %q", args, status, stderr)
		}
	}
	if string(m.Files[wg0Key]) != key {
		t.Error("the key changed")
	}
}

func TestTunnelWG0CheckDueWhereMissing(t *testing.T) {
	p, m := valuesProcess("---\n")
	private, err := wgkey.Generate(bytes.NewReader(bytes.Repeat([]byte{3}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	m.Files[wg0Key] = []byte(private + "\n")
	status, _, stderr := runWs(p, "", "server", "tunnel", "wg0", "--check")
	if status != 1 || !strings.Contains(stderr, "to do: reload networkd for wg0: wg0 is missing") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
	if slices.Contains(m.Calls, "root networkctl reload") {
		t.Errorf("--check reloaded: %q", m.Calls)
	}
}
