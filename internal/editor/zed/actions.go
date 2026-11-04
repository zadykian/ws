// Package zed renders the editors' neutral data as Zed's files. Unbinds, sequences, docks,
// extensions and dap stay here: Zed alone has them.
package zed

import "embed"

//go:embed actions/*.yaml
var actionFiles embed.FS
