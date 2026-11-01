package wizard

import (
	"regexp"

	"github.com/zadykian/ws/mac/internal/effect"
)

// cableStage names the stage in its messages.
const cableStage = "Cable"

// cableEnd is ws's end of the cable, given to its port by the network role. The Mac's service gets
// the other end.
const cableEnd = "10.77.0.1"

// Cable is the stage of the cable from the Mac's display to a port of ws's, outside the router, on a
// machine on the LAN alone.
// Docked, the Mac's cable gives that port carrier, which the files script's link lists among ws's
// wired ports. The stage names the port in host_vars/localhost.yml on ws, whose network role gives
// it 10.77.0.1. Then it names the Mac's service of the cable in mac.conf's WS_CABLE_SERVICE, and
// ws setup gives that 10.77.0.2, and the port in WS_CABLE_PORT, which the stage names on ws again
// after ws's install. Undocked, the first setup waits, and the closing list says to dock the Mac.
type Cable struct {
	Host *HostKey
	Run  *WsRun
	// Root is the stage that logs in as root: ws is read only where it did, in this run.
	Root *RootKey
	// Files runs the files script's operations as root on ws.
	Files *Remote
	// Setup is the ws setup stage, which gives the Mac's service its address.
	Setup *Setup
	// Env looks up a variable of the environment.
	Env func(string) (string, bool)
	P   effect.Prompt
	Tty bool
	M   effect.Machine
	Out effect.Out

	// ports is what link listed on ws at the last read: a line per port.
	ports []cablePort
}

// cablePort is a line of link: a wired port of ws's, its carrier, 1 where a cable plugs in, 0, or -
// where the port is down, and its IPv4 addresses.
type cablePort struct{ name, carrier, addrs string }

var cableEndAt = regexp.MustCompile(`(^| )` + regexp.QuoteMeta(cableEnd) + `/`)

// Name is the stage's name.
func (*Cable) Name() string { return cableStage }

// WaitsOn is ws's run, which makes the clone that holds host_vars on ws.
func (*Cable) WaitsOn() string { return runStage }

type cableChange string

func (c cableChange) String() string { return cableStage + ": would " + string(c) }
