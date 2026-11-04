package fresh

import "embed"

// actionFiles holds the target of each neutral action, or the reason it has none.
//
//go:embed actions/*.yaml
var actionFiles embed.FS
