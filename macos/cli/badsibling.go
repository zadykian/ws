package cli

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/macos/values"
)

// refuseBadSibling refuses where the file of a server other than name, beside mac, is refused.
// remove's steps read the other servers, and a file they skip loses its traces. A bad values.yml
// is refused first and alone: no server is at fault.
func refuseBadSibling(m effect.Machine, mac, name string) error {
	if _, err := values.LoadMac(m, mac); err != nil {
		return err
	}
	names, err := values.Servers(m, mac)
	if err != nil {
		return err
	}
	for _, n := range names {
		if n == name {
			continue
		}
		if _, err := values.Load(m, mac, values.ServerPath(mac, n)); err != nil {
			return values.RefuseSibling(n, err)
		}
	}
	return nil
}
