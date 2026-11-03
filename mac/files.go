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

// Zed is zed.json, the settings that ws mac apply writes as Zed's global_settings.json.
//
//go:embed zed.json
var Zed []byte

// ZedKeymap is zed-keymap.json, the keymap that ws mac apply writes as Zed's keymap.json.
//
//go:embed zed-keymap.json
var ZedKeymap []byte
