package wireguard

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestNetdevListsTheKnownPeers(t *testing.T) {
	got, err := Netdev(51820, []Peer{{"mac", "10.99.0.2", "KEY1"}, {"tv", "10.99.0.3", "KEY2"}})
	if err != nil {
		t.Fatal(err)
	}
	want := "PrivateKeyFile=/etc/systemd/network/60-wg0.key\nListenPort=51820\n\n# mac\n[WireGuardPeer]\n" +
		"PublicKey=KEY1\nAllowedIPs=10.99.0.2/32\n\n# tv\n[WireGuardPeer]\nPublicKey=KEY2\nAllowedIPs=10.99.0.3/32\n"
	if !strings.HasSuffix(string(got), want) {
		t.Errorf("netdev ends:\n%s\nwant it to end with:\n%s", got, want)
	}
}

func TestNetdevWithoutPeersEndsAtThePort(t *testing.T) {
	got, err := Netdev(4000, nil)
	if err != nil || !strings.HasSuffix(string(got), "ListenPort=4000\n") {
		t.Errorf("Netdev = %q, %v", got, err)
	}
}

func TestNetworkGivesWg0ItsAddress(t *testing.T) {
	got, err := Network(values.Wireguard{Address: "10.99.0.1/24"})
	if err != nil || !strings.HasSuffix(string(got), "[Network]\nAddress=10.99.0.1/24\n") {
		t.Errorf("Network = %q, %v", got, err)
	}
}
