// Package ssh reads /etc/ws/ssh_config's text, a Host for each server through the tunnel, which
// ssh's units write, and checks it as ssh reads it.
package ssh

import (
	"fmt"

	"github.com/zadykian/ws/internal/sshcfg"
)

// names is the name of each host.
func names(hosts []sshcfg.Host) []string {
	out := make([]string, len(hosts))
	for i, h := range hosts {
		out[i] = h.Name
	}
	return out
}

// ownHost stops where a Host of the user's own config names a server. The line is counted in
// ~/.ssh/config as it stands.
func ownHost(text string, hosts []sshcfg.Host) error {
	if own := sshcfg.OwnHosts(text, names(hosts)...); len(own) > 0 {
		return fmt.Errorf("line %d of ~/.ssh/config, %s, sets its own values for %s: remove it, as "+
			"/etc/ws/ssh_config holds each server's host", own[0].Line, own[0].Text, own[0].Names)
	}
	return nil
}
