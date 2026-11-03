package security

import "github.com/zadykian/ws/host"

// Where the outbound table and its fill live, in a directory of ws's files alone.
const (
	tableDir   = "/usr/local/lib/ws/security"
	tableFile  = tableDir + "/outbound.nft"
	scriptFile = tableDir + "/resolvers.sh"
)

// The units, in the order they stop: a unit that starts another, as a .path unit, before it.
const (
	pathUnit     = "security-resolvers.path"
	timerUnit    = "security-resolvers.timer"
	fillUnit     = "security-resolvers.service"
	outboundUnit = "security-outbound.service"
)

// paths are what the table's templates name.
type paths struct{ Script, Table string }

// render fills the template files/NAME.tmpl.
func render(name string) ([]byte, error) {
	return host.Render(files, "files/"+name+".tmpl", paths{Script: scriptFile, Table: tableFile})
}
