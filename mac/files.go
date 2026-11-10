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

// ZedSettings is Zed's global_settings.json, which ws mac apply writes whole.
//
//go:embed zed/global_settings.json
var ZedSettings []byte

// ZedKeymap is Zed's keymap.json, which ws mac apply writes whole.
//
//go:embed zed/keymap.json
var ZedKeymap []byte

// Zshrc is ~/.config/ws/zshrc, which ws mac apply writes whole, and which ~/.zshrc sources.
//
//go:embed zshrc
var Zshrc []byte
