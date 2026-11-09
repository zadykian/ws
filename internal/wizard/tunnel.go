package wizard

import (
	"context"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// tunnelStage names the stage in its messages.
const tunnelStage = "Tunnel"

const (
	label = "com.github.zadykian.ws-tunnel"
	// wsEnd is ws's end of the tunnel.
	wsEnd = "10.99.0.1"
)

// Tunnel is the stage of the keys' exchange of the WireGuard tunnel between the Mac and ws. The
// Mac's key goes to wg0's mac.pub on ws, where wg0 takes it as its peer, and ws server apply --only
// wireguard runs there. Both go through the remote script over its direct way, as a change of wg0's peers
// ends the tunnel's sessions. Then mac.conf's WS_PUBLIC_KEY follows wg0's key, and ws mac apply runs
// again, which loads ws-tunnel. Last, ws-tunnel must show a handshake, and ssh ws log in through it.
// Without a handshake, the closing list says so, and the run goes on.
type Tunnel struct {
	// Ws is the session to ws, whose files script names the Mac's key as wg0's peer.
	Ws *Session
	// Conf is mac.conf's path. Check is set under --check.
	Conf  string
	Check bool
	// MacApply runs ws mac apply, as the Setup stage's Run does, again once WS_PUBLIC_KEY is wg0's.
	MacApply func(ctx context.Context, check bool) (bool, error)
	Hand     *Hand
	Home     string
	Sleep    func(time.Duration)
	M        effect.Machine
	Out      effect.Out
	// none is set where the check finds no handshake in this run.
	none bool
}

// Name is the stage's name.
func (*Tunnel) Name() string { return tunnelStage }

// WaitsOn is ws's run, which makes wg0.
func (*Tunnel) WaitsOn() string { return runStage }

type tunnelChange string

func (c tunnelChange) String() string { return tunnelStage + ": would " + string(c) }

// Plan is empty where ws's wg0 has the Mac's key as its peer, WS_PUBLIC_KEY is wg0's key, and
// launchd has ws-tunnel loaded. ws's SSH port must answer through the tunnel too, and ssh ws go
// through it. Where the run asked ws's address, ws isn't at the ways that ws-tunnel tries. Where
// the check found no handshake, ws-tunnel has none. A later run checks those two.
func (t *Tunnel) Plan(ctx context.Context) (step.Plan, error) {
	c, err := t.Ws.want()
	if err != nil {
		return step.Plan{}, err
	}
	mac := t.pub()
	if mac == "" && t.Check {
		t.Out.Say("%s: after ws mac apply", tunnelStage)
		return step.Plan{}, nil
	}
	s, ok := t.Ws.status(ctx)
	if ok && c.PublicKey != "" && s.WgKey == c.PublicKey && s.Peered && mac != "" && s.MacKey == mac &&
		t.loaded(ctx) && (t.Ws.first() != "" || t.none || (t.port(ctx) && t.via(ctx))) {
		return step.Plan{}, nil
	}
	var ch []step.Change
	if s.MacKey != mac {
		ch = append(ch, tunnelChange("write the Mac's key, "+mac+", to /etc/wireguard/mac.pub on ws"))
	}
	if s.MacKey != mac || !s.Peered {
		ch = append(ch, tunnelChange("run ws server apply --only wireguard on ws, for wg0's peer"))
	}
	switch wg0 := s.WgKey; {
	case wg0 == "":
		ch = append(ch, tunnelChange("set WS_PUBLIC_KEY in mac.conf to wg0's key, which that run makes"))
	case wg0 != c.PublicKey:
		if t.Check {
			// Under --check mac.conf is the scratch file, as for the stages before.
			if err := (confFile{t.Conf, tunnelStage, t.M, t.Out}).show("WS_PUBLIC_KEY", wg0); err != nil {
				return step.Plan{}, err
			}
		}
		ch = append(ch, tunnelChange("set WS_PUBLIC_KEY in mac.conf to wg0's key, "+wg0))
	}
	return step.Plan{Changes: append(ch, tunnelChange("run ws mac apply again, which loads ws-tunnel, and wait up "+
		"to 30 s for ws-tunnel's handshake, then log in to ws through it"))}, nil
}
