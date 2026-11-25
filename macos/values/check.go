package values

import (
	"errors"

	"github.com/zadykian/ws/engine/core/effect"
)

// Check reads values.yml at mac and each server's file beside it. It returns every refusal once,
// in its own file's words: a bad key, a missing file, and what two servers share. A server's
// refusal names the server and the way out of its file, as the facts' do. A Mac with no server yet
// has values.yml alone to check.
func Check(m effect.Machine, mac string) error {
	_, err := LoadMac(m, mac)
	names, serversErr := Servers(m, mac)
	all := []error{err, serversErr}
	for _, name := range names {
		server := ServerPath(mac, name)
		c, err := loadServer(m, mac, server)
		if err == nil {
			err = refuseClashes(m, mac, server, c)
		}
		if err != nil {
			err = RefuseSibling(name, err)
		}
		all = append(all, err)
	}
	return errors.Join(all...)
}
