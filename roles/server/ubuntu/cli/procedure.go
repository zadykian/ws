package cli

import enginecli "github.com/zadykian/ws/engine/cli"

// procedure and settle are the shape of a role's own procedure, which a run state runs with
// --check as its check. The Mac's roles take it too, so it lives in engine/cli.
var (
	procedure = enginecli.Procedure
	settle    = enginecli.Settle
)
