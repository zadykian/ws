package tunnel

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/values"
)

// publicKey is a WireGuard public key as wg pubkey prints it: 44 characters of base64.
var publicKey = regexp.MustCompile(`^[A-Za-z0-9+/]{43}=$`)

// checkPeers checks each peer's public key in /etc/wireguard/NAME.pub, a line as wg pubkey prints
// it, in the directory only root reads. wg0's .netdev reads the same files. Access to the machine
// is granted on the machine, as SSH's authorized_keys is: a key in the repository would let its
// peer into every machine it sets up. A peer without the file is left out, and the run names it. A
// file that holds no key is an error.
func checkPeers(m effect.Machine, out effect.Out, w values.Tunnel) error {
	for _, p := range w.Peers {
		path := "/etc/wireguard/" + p.Name + ".pub"
		text, err := m.ReadRootFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			out.Say("wg0 leaves out the peer %s, as %s is missing. Put its public key there, then run the bootstrap again.", p.Name, path)
			continue
		}
		if err != nil {
			return err
		}
		if !publicKey.MatchString(strings.TrimSpace(string(text))) {
			return fmt.Errorf("%s holds no WireGuard public key, 44 characters of base64 as wg pubkey prints them", path)
		}
	}
	return nil
}
