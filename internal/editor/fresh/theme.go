package fresh

import _ "embed"

// Theme is JetBrains Dark as a Fresh theme. A theme is strict JSON: it takes no header.
//
//go:embed themes/jetbrains-dark.json
var Theme []byte
