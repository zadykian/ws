package tunnel

import (
	"fmt"
	"strings"
	"time"
)

// Stats is what the device reports of the peer.
type Stats struct {
	Handshake      time.Time // zero where none
	Received, Sent uint64
	Known          bool // the device has the peer
}

// Format is what ws tunnel status prints, line for line as ws-tunnel status did, as the wizard
// parses it.
func (s State) Format(now time.Time, peer Stats) string {
	var b strings.Builder
	if s.Tier == TierNone {
		b.WriteString("tier: none, as no path to ws answers\n")
	} else {
		fmt.Fprintf(&b, "tier: %s\n", s.Tier)
	}
	fmt.Fprintf(&b, "endpoint: %s\n", orNone(s.Endpoint))
	fmt.Fprintf(&b, "interface: %s\n", s.Interface)
	fmt.Fprintf(&b, "since: %d s ago\n", ago(now, s.Since))
	if peer.Handshake.IsZero() {
		b.WriteString("latest handshake: none\n")
	} else {
		fmt.Fprintf(&b, "latest handshake: %d s ago\n", ago(now, peer.Handshake))
	}
	if peer.Known {
		fmt.Fprintf(&b, "transfer: %d B received, %d B sent\n", peer.Received, peer.Sent)
	}
	fmt.Fprintf(&b, "routes: %s\n", orNone(joinNets(s.Routes)))
	if len(s.LeftOut) > 0 {
		fmt.Fprintf(&b, "left out, as they hold the endpoint: %s\n", joinNets(s.LeftOut))
	}
	return b.String()
}

func ago(now, then time.Time) int64 {
	return now.Unix() - then.Unix()
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
