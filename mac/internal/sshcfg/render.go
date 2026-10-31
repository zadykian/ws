// Package sshcfg holds the text of ~/.ssh/config.d/ws and the reading of ssh_config files, all pure.
package sshcfg

import (
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/tunnel"
)

// Header starts config.d/ws's first line, by which a run knows a file for its own.
const Header = "# Written by ws's mac/setup.sh"

// Inputs are what config.d/ws's text depends on.
type Inputs struct {
	User       string
	LAN        string // WS_LAN, or empty
	LANAddress string
	Remote     string
	RemotePort string // empty where mac.conf has no forward
	// ViaTunnel is whether ws goes through the tunnel. Until it does, ws takes the ways of ws-ssh.
	ViaTunnel bool
	// Patterns are the IgnoreUnknown patterns, joined by commas.
	Patterns string
	// IdentityPub is the .pub of the key ssh offers ws, as the file names it.
	IdentityPub string
	// IdentityAgent is that key's agent socket, or empty for a key in a file.
	IdentityAgent string
}

// Config returns config.d/ws. ssh keeps the first value it reads for a key, so each Match sets Port
// 22 before Host ws-ssh sets the forward's port, and the cable comes before the LAN. localnetwork
// matches the Mac's own addresses, with no probe. originalhost matches the name typed, which a
// HostName set above does not change, unlike Match host.
func Config(in Inputs) string {
	direct, listed := "ws-ssh", "ws-ssh"
	if !in.ViaTunnel {
		direct, listed = "ws ws-ssh", "ws,ws-ssh"
	}
	var b strings.Builder
	fmt.Fprintf(&b, `%s from ~/.config/ws/mac.conf: its next run replaces changes made here.

# Only Apple's ssh knows UseKeychain. ssh keeps the first IgnoreUnknown it reads, this one, so it
# carries the patterns of ~/.ssh/config's own too: run setup.sh again after a change to those.
IgnoreUnknown %s

# ws over the cable, where the Mac has an address on it.
Match originalhost %s localnetwork %s
    HostName %s
    Port 22
`, Header, in.Patterns, listed, tunnel.CableNet, tunnel.CableServer)
	if in.LAN != "" {
		fmt.Fprintf(&b, `# ws on the home LAN, where the Mac has an address there.
Match originalhost %s localnetwork %s
    HostName %s
    Port 22
`, listed, in.LAN, in.LANAddress)
	}
	if in.RemotePort != "" {
		fmt.Fprintf(&b, `# Elsewhere, at WS_REMOTE, through the router's port forward or to a cloud VM, over IPv4 alone, as
# a forward takes it, while the name may have an AAAA record too.
Host %s
    HostName %s
    Port %s
    AddressFamily inet
`, direct, in.Remote, in.RemotePort)
	}
	if in.ViaTunnel {
		fmt.Fprintf(&b, `
# ws through the tunnel, at the one address it has there whichever way the Mac is connected, on
# ws's own port, whatever a Host * of the user's sets.
Host ws
    HostName %s
    Port 22
`, tunnel.Server)
	}
	fmt.Fprintf(&b, `
# One known_hosts entry, ws's, whichever way ssh goes. 30 s between checks, 3 of them unanswered
# before ssh gives up.
Host ws ws-ssh
    User %s
    HostKeyAlias ws
%s    ServerAliveInterval 30
`, in.User, identityLines(in))
	return b.String()
}

// identityLines are the lines for the key that ssh offers ws alone: with an agent, the agent's key
// that matches the .pub. Else the file beside the .pub, whose passphrase the Keychain keeps, and
// which the agent takes once ssh has read it.
func identityLines(in Inputs) string {
	if in.IdentityAgent != "" {
		return fmt.Sprintf("    IdentityFile %s\n    IdentitiesOnly yes\n    IdentityAgent \"%s\"\n",
			in.IdentityPub, in.IdentityAgent)
	}
	return fmt.Sprintf("    IdentityFile %s\n    IdentitiesOnly yes\n    UseKeychain yes\n    AddKeysToAgent yes\n",
		strings.TrimSuffix(in.IdentityPub, ".pub"))
}
