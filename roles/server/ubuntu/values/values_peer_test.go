package values

import "testing"

func TestParseListsTheMacAtWg0sAddressPlusOne(t *testing.T) {
	for in, want := range map[string]string{
		"":                                   "10.99.0.2",
		"wireguard_address: 10.99.1.1/24\n":  "10.99.1.2",
		"wireguard_address: 10.99.15.1/24\n": "10.99.15.2",
	} {
		got, err := Parse([]byte(in))
		if err != nil || len(got.Wireguard.Peers) != 1 || got.Wireguard.Peers[0] != (WireguardPeer{Name: "mac", Address: want}) {
			t.Errorf("Parse(%q) = %v, %v; want mac at %s", in, got.Wireguard.Peers, err, want)
		}
	}
}

func TestParseKeepsThePeersItIsGiven(t *testing.T) {
	got, err := Parse([]byte("wireguard_peers: []\n"))
	if err != nil || len(got.Wireguard.Peers) != 0 {
		t.Errorf("an empty list gave %v, %v; want none", got.Wireguard.Peers, err)
	}
	got, err = Parse([]byte("wireguard_peers:\n  - {name: ipad, address: 10.99.0.9}\n"))
	if err != nil || len(got.Wireguard.Peers) != 1 || got.Wireguard.Peers[0].Name != "ipad" {
		t.Errorf("a given peer gave %v, %v; want ipad alone", got.Wireguard.Peers, err)
	}
}

func TestParseListsNoMacForAnAddressItCannotRead(t *testing.T) {
	got, err := Parse([]byte("wireguard_address: nope\n"))
	if err != nil || got.Wireguard.Peers != nil {
		t.Errorf("Parse = %v, %v; want no peer", got.Wireguard.Peers, err)
	}
}
