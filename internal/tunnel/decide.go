package tunnel

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"
)

// Decide chooses the way to ws at the time now, from the Mac's addresses, the settings, the name
// as resolved and the peer. Between rounds it keeps the endpoint in use, unless the settings or the
// Mac's addresses changed, no path answered, or ws missed pings. Then a round tries each endpoint
// in order, waits for a handshake from it within Patience, and moves on where none comes, or where
// it comes from another endpoint, as one that ws started over another path proves nothing.
func Decide(now time.Time, ifaces []Iface, c Conf, resolved netip.Addr, p Peer) Decision {
	found, missing := Candidates(ifaces, c, resolved)
	if len(p.Round) == 0 {
		why := retryReason(ifaces, c, p)
		if why == "" {
			return Decision{Keep, p.InUse, ""}
		}
		return next(found, missing, p.Round, c.Port, why)
	}
	last := p.Round[len(p.Round)-1]
	want := netip.AddrPortFrom(last.Candidate.Host, c.Port)
	// WireGuard counts the handshake in whole seconds, which may be a hair before the attempt.
	if !p.Handshake.IsZero() && !p.Handshake.Before(last.At.Truncate(time.Second)) {
		if p.Endpoint == want {
			return Decision{Keep, last.Candidate, fmt.Sprintf("on the %s tier, %s", last.Candidate.Tier, want)}
		}
		came := fmt.Sprintf("the handshake came from %s, not %s", p.Endpoint, want)
		dec := next(found, missing, p.Round, c.Port, came)
		if dec.Action == Fail {
			// The last endpoint is the one that failed, and the log keeps what the shell said of it.
			dec.Why = came + "; " + dec.Why
		}
		return dec
	}
	if now.Sub(last.At) < Patience {
		return Decision{Wait, last.Candidate, ""}
	}
	return next(found, missing, p.Round, c.Port, "")
}

// The reasons for a new round that the daemon does not log, as they are no news.
const (
	WhyAddresses = "the Mac's addresses changed"
	WhyNone      = "no path answered"
)

// retryReason says why the paths are tried again between rounds, or is empty.
func retryReason(ifaces []Iface, c Conf, p Peer) string {
	switch {
	case p.Conf != c:
		return "read new settings; trying the paths again"
	case !slices.Equal(ifaces, p.Seen):
		return WhyAddresses
	case !p.InUse.Host.IsValid():
		return WhyNone
	case p.Misses >= MissedPings:
		return fmt.Sprintf("ws missed %d pings in a row; trying the paths again", MissedPings)
	}
	return ""
}

// next tries the first endpoint that the round has not, or fails where none is left.
func next(found []Candidate, missing []string, round []Attempt, port uint16, why string) Decision {
	var failed []string
	for _, each := range found {
		if !slices.ContainsFunc(round, func(a Attempt) bool { return a.Candidate == each }) {
			return Decision{Try, each, why}
		}
	}
	for _, a := range round {
		failed = append(failed, fmt.Sprintf("no handshake from %s on the %s tier",
			netip.AddrPortFrom(a.Candidate.Host, port), a.Candidate.Tier))
	}
	failed = append(failed, missing...)
	reason := "as the Mac is on none"
	if len(failed) > 0 {
		reason = strings.Join(failed, ", ")
	}
	return Decision{Fail, Candidate{}, "no path to ws answers, " + reason}
}
