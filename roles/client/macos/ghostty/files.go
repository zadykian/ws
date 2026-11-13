package ghostty

import _ "embed" // for the file's go:embed

// Settings is ws.ghostty, the settings that ws client macos apply includes in Ghostty's config.
//
//go:embed files/ws.ghostty
var Settings []byte
