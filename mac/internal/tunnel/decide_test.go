package tunnel

import (
	"net/netip"
	"testing"
	"time"
)

func TestDecide(t *testing.T) {
	t0 := time.Date(2026, 10, 31, 12, 0, 0, 500_000_000, time.UTC)
	conf := Conf{Key: "k", Port: 51820, Remote: "ws.example.com",
		LAN: netip.MustParsePrefix("192.168.1.0/24"), LANServer: netip.MustParseAddr("192.168.1.10")}
	home := []Iface{{"en0", netip.MustParseAddr("192.168.1.50")}}
	away := []Iface{{"en0", netip.MustParseAddr("172.20.0.5")}}
	ws := netip.MustParseAddr("203.0.113.9")
	lan := Candidate{TierLAN, conf.LANServer}
	remote := Candidate{TierRemote, ws}
	steady := Peer{InUse: lan, Conf: conf, Seen: home}
	round := func(a ...Candidate) []Attempt {
		var out []Attempt
		for i, each := range a {
			out = append(out, Attempt{each, t0.Add(time.Duration(i) * Patience)})
		}
		return out
	}
	with := func(p Peer, f func(*Peer)) Peer { f(&p); return p }
	lanPort := netip.AddrPortFrom(lan.Host, 51820)
	tests := []struct {
		name   string
		now    time.Time
		ifaces []Iface
		peer   Peer
		want   Decision
	}{
		{"steady", t0, home, steady, Decision{Keep, lan, ""}},
		{"two pings missed", t0, home, with(steady, func(p *Peer) { p.Misses = 2 }), Decision{Keep, lan, ""}},
		{"three pings missed", t0, home, with(steady, func(p *Peer) { p.Misses = 3 }),
			Decision{Try, lan, "ws missed 3 pings in a row; trying the paths again"}},
		{"new settings", t0, home, with(steady, func(p *Peer) { p.Conf.Port = 1 }),
			Decision{Try, lan, "read new settings; trying the paths again"}},
		{"addresses changed", t0, away, steady, Decision{Try, remote, "the Mac's addresses changed"}},
		{"no path in use", t0, home, with(steady, func(p *Peer) { p.InUse = Candidate{} }),
			Decision{Try, lan, "no path answered"}},
		{"waits for the handshake", t0.Add(4 * time.Second), home, with(steady, func(p *Peer) { p.Round = round(lan) }),
			Decision{Wait, lan, ""}},
		{"a stale handshake waits", t0.Add(4 * time.Second), home, with(steady, func(p *Peer) {
			p.Round = round(lan)
			p.Handshake = t0.Add(-time.Minute)
		}), Decision{Wait, lan, ""}},
		{"the handshake comes", t0.Add(2 * time.Second), home, with(steady, func(p *Peer) {
			p.Round, p.Endpoint = round(lan), lanPort
			p.Handshake = t0.Truncate(time.Second)
		}), Decision{Keep, lan, "on the lan tier, 192.168.1.10:51820"}},
		{"the handshake is from another endpoint", t0.Add(2 * time.Second), home, with(steady, func(p *Peer) {
			p.Round, p.Endpoint = round(lan), netip.MustParseAddrPort("203.0.113.9:51820")
			p.Handshake = t0
		}), Decision{Try, remote, "the handshake came from 203.0.113.9:51820, not 192.168.1.10:51820"}},
		{"the handshake is from another endpoint, and none is left", t0.Add(2 * time.Second), home, with(steady, func(p *Peer) {
			p.Round, p.Endpoint = round(lan, remote), lanPort
			p.Handshake = t0.Add(Patience)
		}), Decision{Fail, Candidate{}, "the handshake came from 192.168.1.10:51820, not 203.0.113.9:51820; " +
			"no path to ws answers, no handshake from 192.168.1.10:51820 on the lan tier, " +
			"no handshake from 203.0.113.9:51820 on the remote tier"}},
		{"patience ends", t0.Add(Patience), home, with(steady, func(p *Peer) { p.Round = round(lan) }),
			Decision{Try, remote, ""}},
		{"every endpoint failed", t0.Add(2 * Patience), home, with(steady, func(p *Peer) { p.Round = round(lan, remote) }),
			Decision{Fail, Candidate{}, "no path to ws answers, no handshake from 192.168.1.10:51820 on the lan tier, " +
				"no handshake from 203.0.113.9:51820 on the remote tier"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Decide(tc.now, tc.ifaces, conf, ws, tc.peer); got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestDecideOnNoPath(t *testing.T) {
	p := Peer{Conf: Conf{Key: "k", Port: 1}, Seen: nil}
	got := Decide(time.Time{}, nil, p.Conf, netip.Addr{}, p)
	if got.Action != Fail || got.Why != "no path to ws answers, as the Mac is on none" {
		t.Errorf("got %+v", got)
	}
}
