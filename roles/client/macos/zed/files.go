package zed

import _ "embed" // for the files' go:embed

// Settings is Zed's global_settings.json, which ws client macos apply writes whole.
//
//go:embed files/global_settings.json
var Settings []byte

// Keymap is Zed's keymap.json, which ws client macos apply writes whole.
//
//go:embed files/keymap.json
var Keymap []byte
