package wireguard

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/server/values"
)

// publicKey is a WireGuard public key as wg pubkey prints it: 44 characters of base64.
var publicKey = regexp.MustCompile(`^[A-Za-z0-9+/]{43}=$`)

// peerKeys reads each peer's public key from /etc/wireguard/NAME.pub, a line as wg pubkey prints
// it, in the directory only root reads. Access to the machine is granted on the machine, as SSH's
// authorized_keys is: a key in the repository would let its peer into every machine it sets up. A
// peer without the file is left out, and the run names it. A file that holds no key is an error.
func peerKeys(m effect.Machine, out effect.Out, w values.Wireguard) ([]Peer, error) {
	var peers []Peer
	for _, p := range w.Peers {
		path := "/etc/wireguard/" + p.Name + ".pub"
		text, err := m.ReadRootFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			effect.SayLasting(out, "wg0 leaves out the peer %s, as %s is missing. Put its public key there, then run ws server apply --only wireguard again.", p.Name, path)
			continue
		}
		if err != nil {
			return nil, err
		}
		key := strings.TrimSpace(string(text))
		if !publicKey.MatchString(key) {
			return nil, fmt.Errorf("%s holds no WireGuard public key, 44 characters of base64 as wg pubkey prints them", path)
		}
		peers = append(peers, Peer{p.Name, p.Address, key})
	}
	return peers, nil
}
