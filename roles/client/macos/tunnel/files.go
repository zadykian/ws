package tunnel

import _ "embed" // for the file's go:embed

// Plist is the LaunchDaemon's plist, which ws client macos apply installs for the tunnel.
//
//go:embed files/com.github.zadykian.ws-tunnel.plist
var Plist []byte
