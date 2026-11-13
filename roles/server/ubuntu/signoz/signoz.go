// Package signoz sets up SigNoz in Docker from foundryctl's compose files, and sets its retention
// and dashboards through its API. The compose files, the casting they are forged from and the
// dashboards are embedded from this directory.
package signoz

import "embed"

// deployment holds foundryctl's forged files. They are never edited by hand: change casting.yaml
// and forge again.
//
//go:embed deployment
var deployment embed.FS

// Project is the compose project's name, which compose.yaml sets.
const Project = "signoz"
