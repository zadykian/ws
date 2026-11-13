package tunnel

import _ "embed" // for the file's go:embed

// Plist is the template of the LaunchDaemon's plist, which ws client macos apply installs for the
// tunnel of each server, with @NAME@ the server's name.
//
//go:embed files/ws-tunnel.plist
var Plist []byte
