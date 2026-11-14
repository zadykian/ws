package cli

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// refuseBadSibling refuses where the file of a server other than name, beside mac, is refused.
// Apply's and remove's steps read the other servers, and a file they skip loses its traces. The
// networks step removes the resolver files of a server it cannot read.
func refuseBadSibling(m effect.Machine, mac, name string) error {
	names, err := conf.Servers(m, mac)
	if err != nil {
		return err
	}
	for _, n := range names {
		if n == name {
			continue
		}
		if _, err := conf.Load(m, mac, conf.ServerPath(mac, n)); err != nil {
			return conf.RefuseSibling(n, err)
		}
	}
	return nil
}
