package wireguard

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
)

const keyA = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestPeerKeysReadTheFilesAndNameTheMissing(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Files["/etc/wireguard/mac.pub"] = []byte(keyA + "\n")
	w := values.Wireguard{Peers: []values.WireguardPeer{{Name: "mac", Address: "10.99.0.2"}, {Name: "tv", Address: "10.99.0.3"}}}
	got, err := peerKeys(m, out, w)
	if err != nil || len(got) != 1 || got[0] != (Peer{"mac", "10.99.0.2", keyA}) {
		t.Fatalf("peerKeys = %v, %v", got, err)
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "wg0 leaves out the peer tv, as /etc/wireguard/tv.pub is missing") {
		t.Errorf("output = %q", out.Lines)
	}
}

func TestPeerKeysRefuseAFileWithNoKey(t *testing.T) {
	for name, content := range map[string]string{
		"empty": "", "short": "AAAA=\n", "no padding": strings.Repeat("A", 44), "a private key's file with two lines": keyA + "\n" + keyA,
	} {
		t.Run(name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Files["/etc/wireguard/mac.pub"] = []byte(content)
			w := values.Wireguard{Peers: []values.WireguardPeer{{Name: "mac", Address: "10.99.0.2"}}}
			_, err := peerKeys(m, &effecttest.Out{}, w)
			if err == nil || !strings.Contains(err.Error(), "/etc/wireguard/mac.pub holds no WireGuard public key") {
				t.Errorf("peerKeys = %v, want a refusal", err)
			}
		})
	}
}
