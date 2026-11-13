package zsh

import _ "embed" // for the file's go:embed

// Hook is ~/.config/ws/zshrc, which ws client macos apply writes whole, and which ~/.zshrc sources.
//
//go:embed files/zshrc
var Hook []byte
