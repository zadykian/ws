package values

import (
	"strings"
	"testing"
)

func TestParseListsTheMacAtWg0sAddressPlusOne(t *testing.T) {
	for in, want := range map[string]string{
		"":                                   "10.99.0.2",
		"WS_TUNNEL_ADDRESS: 10.99.1.1/24\n":  "10.99.1.2",
		"WS_TUNNEL_ADDRESS: 10.99.15.1/24\n": "10.99.15.2",
	} {
		got, err := Parse([]byte(in))
		if err != nil || len(got.Tunnel.Peers) != 1 || got.Tunnel.Peers[0] != (TunnelPeer{Name: "mac", Address: want}) {
			t.Errorf("Parse(%q) = %v, %v; want mac at %s", in, got.Tunnel.Peers, err, want)
		}
	}
}

func TestParseKeepsThePeersItIsGiven(t *testing.T) {
	got, err := Parse([]byte("WS_TUNNEL_PEERS: []\n"))
	if err != nil || len(got.Tunnel.Peers) != 0 {
		t.Errorf("an empty list gave %v, %v; want none", got.Tunnel.Peers, err)
	}
	got, err = Parse([]byte("WS_TUNNEL_PEERS:\n  - {name: ipad, address: 10.99.0.9}\n"))
	if err != nil || len(got.Tunnel.Peers) != 1 || got.Tunnel.Peers[0].Name != "ipad" {
		t.Errorf("a given peer gave %v, %v; want ipad alone", got.Tunnel.Peers, err)
	}
}

func TestDecodeListsNoMacForAnAddressItCannotRead(t *testing.T) {
	got, err := decode([]byte("WS_TUNNEL_ADDRESS: nope\n"))
	if err != nil || got.Tunnel.Peers != nil {
		t.Errorf("decode = %v, %v; want no peer", got.Tunnel.Peers, err)
	}
}

func TestParseRefusesABadTunnelPort(t *testing.T) {
	if _, err := Parse([]byte("WS_TUNNEL_PORT: 0\n")); err == nil || !strings.Contains(err.Error(), "WS_TUNNEL_PORT is 0") {
		t.Errorf("Parse = %v, want the port refused", err)
	}
}

func TestParseTakesANullPeerListAsNoPeer(t *testing.T) {
	for _, in := range []string{"WS_TUNNEL_PEERS:\n", "WS_TUNNEL_PEERS: ~\n", "WS_TUNNEL_PEERS: null\n", "WS_TUNNEL_ADDRESS: 10.98.0.1/24\nWS_TUNNEL_PEERS:\n"} {
		got, err := Parse([]byte(in))
		if err != nil || len(got.Tunnel.Peers) != 0 {
			t.Errorf("Parse(%q) = %v, %v; want no peer", in, got.Tunnel.Peers, err)
		}
	}
}
