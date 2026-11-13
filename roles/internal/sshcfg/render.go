// Package sshcfg holds the text of ~/.ssh/config.d/ws, the reading of ssh_config files and of a
// .pub's line, all pure.
package sshcfg

import (
	"cmp"
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// Header starts config.d/ws's first line, by which a run knows a file for its own.
const Header = "# Written by ws client macos apply"

// Host is one server's block.
type Host struct {
	// Name is the server's name: its Host, and the alias of its key in known_hosts.
	Name string
	// Addr is the tunnel's end, the server's address on it.
	Addr netip.Addr
	User string
}

// Inputs are what config.d/ws's text depends on.
type Inputs struct {
	// Hosts are the servers, each given a block, in name order.
	Hosts []Host
	// CldTabs is whether ssh's hosts start cld's tab restorer.
	CldTabs bool
	// Patterns are the IgnoreUnknown patterns, joined by commas.
	Patterns string
	// IdentityPub is the .pub of the key ssh offers ws, as the file names it.
	IdentityPub string
	// IdentityAgent is that key's agent socket, or empty for a key in a file.
	IdentityAgent string
}

// Config returns config.d/ws: a Host for each server, at the tunnel's end. The tunnel is the one
// address a server has, whichever way the Mac is connected. Port 22 is the server's own, whatever
// a Host * of the user's sets.
func Config(in Inputs) string {
	var b strings.Builder
	fmt.Fprintf(&b, `%s from ~/.config/ws/mac.conf: its next run replaces changes made here.

# Only Apple's ssh knows UseKeychain. ssh keeps the first IgnoreUnknown it reads, this one, so it
# carries the patterns of ~/.ssh/config's own too: run ws client macos apply again after a change to
# those.
IgnoreUnknown %s

`, Header, in.Patterns)
	hosts := slices.SortedFunc(slices.Values(in.Hosts), func(a, b Host) int { return cmp.Compare(a.Name, b.Name) })
	for i, h := range hosts {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, `# %[1]s through the tunnel. One known_hosts entry, %[1]s's. 30 s between checks, 3 of them unanswered
# before ssh gives up.
Host %[1]s
    HostName %[2]s
    Port 22
    User %[3]s
    HostKeyAlias %[1]s
%[4]s    ServerAliveInterval 30
`, h.Name, h.Addr, h.User, identityLines(in))
		if in.CldTabs {
			b.WriteString(cldLines)
		}
	}
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

// cldLines end the last block with the lines that start cld's tab restorer. ssh runs LocalCommand
// through the user's shell once it has connected, with %d as the Mac's home (ssh_config(5)). The
// line starts it in a Ghostty pane alone, so that macOS charges its Apple Events to Ghostty.
// Elsewhere, or without cld, it does nothing and exits 0.
const cldLines = `    # In a Ghostty pane, start cld's watcher, unless one runs, for cld restore tabs.
    PermitLocalCommand yes
    LocalCommand [ "$TERM_PROGRAM" != ghostty ] || [ ! -x %d/.local/bin/cld ] || %d/.local/bin/cld restore tabs --watch
`
