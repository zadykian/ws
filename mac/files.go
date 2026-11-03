package mac

import _ "embed" // for the files' go:embed

// Ghostty is ws.ghostty, the settings that ws mac apply includes in Ghostty's config.
//
//go:embed ws.ghostty
var Ghostty []byte

// TunnelPlist is the LaunchDaemon's plist, which ws mac apply installs for the tunnel.
//
//go:embed com.github.zadykian.ws-tunnel.plist
var TunnelPlist []byte
