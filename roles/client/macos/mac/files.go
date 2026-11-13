// Package mac holds the files that the ws CLI embeds for the Mac: the tunnel's plist, Zed's
// settings and the zsh hook.
package mac

import _ "embed" // for the files' go:embed

// TunnelPlist is the LaunchDaemon's plist, which ws client macos apply installs for the tunnel.
//
//go:embed com.github.zadykian.ws-tunnel.plist
var TunnelPlist []byte

// ZedSettings is Zed's global_settings.json, which ws client macos apply writes whole.
//
//go:embed zed/global_settings.json
var ZedSettings []byte

// ZedKeymap is Zed's keymap.json, which ws client macos apply writes whole.
//
//go:embed zed/keymap.json
var ZedKeymap []byte

// Zshrc is ~/.config/ws/zshrc, which ws client macos apply writes whole, and which ~/.zshrc
// sources.
//
//go:embed zshrc
var Zshrc []byte
