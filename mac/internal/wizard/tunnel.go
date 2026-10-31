package wizard

import (
	"time"

	"github.com/zadykian/ws/mac/internal/effect"
)

// tunnelStage names the stage in its messages.
const tunnelStage = "Tunnel"

const (
	label = "com.github.zadykian.ws-tunnel"
	// wsEnd is ws's end of the tunnel.
	wsEnd = "10.99.0.1"
)

// Tunnel is the stage of the keys' exchange of the WireGuard tunnel between the Mac and ws. The
// Mac's key goes to wg0's mac.pub on ws, where wg0 takes it as its peer, and bootstrap.sh --tags
// wireguard runs there. Both go through the remote script over ws-ssh, as a change of wg0's peers
// ends the tunnel's sessions. Then mac.conf's WS_PUBLIC_KEY follows wg0's key, and ws setup runs
// again, which loads ws-tunnel. Last, ws-tunnel must show a handshake, and ssh ws log in through it.
// Without a handshake, the closing list says so, and the run goes on.
type Tunnel struct {
	Host *HostKey
	Run  *WsRun
	// Setup is the ws setup stage, which runs again once WS_PUBLIC_KEY is wg0's.
	Setup *Setup
	Hand  *Hand
	Home  string
	Sleep func(time.Duration)
	M     effect.Machine
	Out   effect.Out
	// none is set where the check finds no handshake in this run.
	none bool
}

// Name is the stage's name.
func (*Tunnel) Name() string { return tunnelStage }

// WaitsOn is ws's run, which makes wg0.
func (*Tunnel) WaitsOn() string { return runStage }
