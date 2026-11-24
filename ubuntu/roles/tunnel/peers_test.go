package tunnel

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

const keyA = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestCheckPeersNamesTheMissing(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Files["/etc/wireguard/mac.pub"] = []byte(keyA + "\n")
	w := values.Tunnel{Peers: []values.TunnelPeer{{Name: "mac", Address: "10.99.0.2"}, {Name: "tv", Address: "10.99.0.3"}}}
	if err := checkPeers(m, out, w); err != nil {
		t.Fatalf("checkPeers = %v", err)
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "wg0 leaves out the peer tv, as /etc/wireguard/tv.pub is missing") {
		t.Errorf("output = %q", out.Lines)
	}
}

func TestCheckPeersRefusesAFileWithNoKey(t *testing.T) {
	for name, content := range map[string]string{
		"empty": "", "short": "AAAA=\n", "no padding": strings.Repeat("A", 44), "a private key's file with two lines": keyA + "\n" + keyA,
	} {
		t.Run(name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Files["/etc/wireguard/mac.pub"] = []byte(content)
			w := values.Tunnel{Peers: []values.TunnelPeer{{Name: "mac", Address: "10.99.0.2"}}}
			err := checkPeers(m, &effecttest.Out{}, w)
			if err == nil || !strings.Contains(err.Error(), "/etc/wireguard/mac.pub holds no WireGuard public key") {
				t.Errorf("checkPeers = %v, want a refusal", err)
			}
		})
	}
}
