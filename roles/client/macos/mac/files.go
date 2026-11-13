// Package mac holds the file that the ws CLI embeds for the Mac: the tunnel's plist.
package mac

import _ "embed" // for the file's go:embed

// TunnelPlist is the LaunchDaemon's plist, which ws client macos apply installs for the tunnel.
//
//go:embed com.github.zadykian.ws-tunnel.plist
var TunnelPlist []byte
