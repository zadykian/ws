package values

import "testing"

func TestParseListsTheMacAtWg0sAddressPlusOne(t *testing.T) {
	for in, want := range map[string]string{
		"":                                   "10.99.0.2",
		"WS_TUNNEL_ADDRESS: 10.99.1.1/24\n":  "10.99.1.2",
		"WS_TUNNEL_ADDRESS: 10.99.15.1/24\n": "10.99.15.2",
	} {
		got, err := Parse([]byte(in))
		if err != nil || len(got.Wireguard.Peers) != 1 || got.Wireguard.Peers[0] != (WireguardPeer{Name: "mac", Address: want}) {
			t.Errorf("Parse(%q) = %v, %v; want mac at %s", in, got.Wireguard.Peers, err, want)
		}
	}
}

func TestParseKeepsThePeersItIsGiven(t *testing.T) {
	got, err := Parse([]byte("WS_TUNNEL_PEERS: []\n"))
	if err != nil || len(got.Wireguard.Peers) != 0 {
		t.Errorf("an empty list gave %v, %v; want none", got.Wireguard.Peers, err)
	}
	got, err = Parse([]byte("WS_TUNNEL_PEERS:\n  - {name: ipad, address: 10.99.0.9}\n"))
	if err != nil || len(got.Wireguard.Peers) != 1 || got.Wireguard.Peers[0].Name != "ipad" {
		t.Errorf("a given peer gave %v, %v; want ipad alone", got.Wireguard.Peers, err)
	}
}

func TestParseListsNoMacForAnAddressItCannotRead(t *testing.T) {
	got, err := Parse([]byte("WS_TUNNEL_ADDRESS: nope\n"))
	if err != nil || got.Wireguard.Peers != nil {
		t.Errorf("Parse = %v, %v; want no peer", got.Wireguard.Peers, err)
	}
}

func TestParseTakesANullPeerListAsNoPeer(t *testing.T) {
	for _, in := range []string{"WS_TUNNEL_PEERS:\n", "WS_TUNNEL_PEERS: ~\n", "WS_TUNNEL_PEERS: null\n", "WS_TUNNEL_ADDRESS: 10.98.0.1/24\nWS_TUNNEL_PEERS:\n"} {
		got, err := Parse([]byte(in))
		if err != nil || len(got.Wireguard.Peers) != 0 {
			t.Errorf("Parse(%q) = %v, %v; want no peer", in, got.Wireguard.Peers, err)
		}
	}
}
