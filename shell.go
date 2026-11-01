package ws

import _ "embed" // for the defaults' go:embed

// ShellDefaults is the shell role's defaults, whose git identity the wizard offers as answers.
//
//go:embed roles/shell/defaults/main.yml
var ShellDefaults []byte
