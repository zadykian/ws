// Package sshcfg holds the text of ~/.ssh/config.d/ws, the reading of ssh_config files and of a
// .pub's line, all pure.
package sshcfg

import (
	"fmt"
	"strings"

	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

// Header starts config.d/ws's first line, by which a run knows a file for its own.
const Header = "# Written by ws mac apply"

// Inputs are what config.d/ws's text depends on.
type Inputs struct {
	User string
	// CldTabs is whether ssh's hosts start cld's tab restorer.
	CldTabs bool
	// Patterns are the IgnoreUnknown patterns, joined by commas.
	Patterns string
	// IdentityPub is the .pub of the key ssh offers ws, as the file names it.
	IdentityPub string
	// IdentityAgent is that key's agent socket, or empty for a key in a file.
	IdentityAgent string
}

// Config returns config.d/ws: one Host ws, at the tunnel's end. The tunnel is the one address ws
// has whichever way the Mac is connected, and Port 22 is ws's own, whatever a Host * of the
// user's sets.
func Config(in Inputs) string {
	var b strings.Builder
	fmt.Fprintf(&b, `%s from ~/.config/ws/mac.conf: its next run replaces changes made here.

# Only Apple's ssh knows UseKeychain. ssh keeps the first IgnoreUnknown it reads, this one, so it
# carries the patterns of ~/.ssh/config's own too: run ws mac apply again after a change to those.
IgnoreUnknown %s

# ws through the tunnel. One known_hosts entry, ws's. 30 s between checks, 3 of them unanswered
# before ssh gives up.
Host ws
    HostName %s
    Port 22
    User %s
    HostKeyAlias ws
%s    ServerAliveInterval 30
`, Header, in.Patterns, tunnel.Server, in.User, identityLines(in))
	if in.CldTabs {
		b.WriteString(cldLines)
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
