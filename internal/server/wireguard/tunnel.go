package wireguard

import (
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

const (
	// keyFile is the server's private key, which networkd reads for wg0.
	keyFile    = "/etc/systemd/network/60-wg0.key"
	netdevFile = "/etc/systemd/network/60-wg0.netdev"
	// networkFile is wg0's address, and no gateway, DNS or route beyond its subnet's.
	networkFile = "/etc/systemd/network/60-wg0.network"
)

// Peer is a peer of wg0 by its address in the tunnel and its public key.
type Peer struct{ Name, Address, Key string }

// Netdev returns wg0's .netdev, which lists the peers given: those whose key is known.
func Netdev(port int, peers []Peer) ([]byte, error) {
	return host.Render(files, "files/60-wg0.netdev.tmpl", struct {
		KeyFile string
		Port    int
		Peers   []Peer
	}{keyFile, port, peers})
}

// Network returns wg0's .network, which gives it its address with the tunnel's prefix length.
func Network(w values.Wireguard) ([]byte, error) {
	return host.Render(files, "files/60-wg0.network.tmpl", w)
}
