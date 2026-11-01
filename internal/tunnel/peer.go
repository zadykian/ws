package tunnel

import (
	"net/netip"
	"time"
)

// Patience is how long a handshake has to complete with an endpoint, WireGuard's own retry time.
const Patience = 5 * time.Second

// MissedPings is how many pings in a row ws may miss before the daemon tries the paths again.
const MissedPings = 3

// Attempt is an endpoint that the current round of choices tried, and when.
type Attempt struct {
	Candidate Candidate
	At        time.Time
}

// Peer is what the daemon knows when it chooses.
type Peer struct {
	InUse     Candidate // the endpoint in use, or none where Host is invalid
	Conf      Conf      // the settings that the last choice used
	Seen      []Iface   // the Mac's addresses at the last choice
	Misses    int       // pings of ws missed in a row
	Round     []Attempt // the endpoints tried since the round began, empty between rounds
	Handshake time.Time // the peer's latest handshake, zero where none
	Endpoint  netip.AddrPort
}

// Action is what the daemon does next.
type Action int

// The actions.
const (
	Keep Action = iota // the endpoint in use stands
	Wait               // the endpoint tried has time left to answer
	Try                // point the peer at Decision.Candidate
	Fail               // no endpoint answers
)

// Decision is the choice of a way, and why.
type Decision struct {
	Action    Action
	Candidate Candidate
	Why       string
}
