package conf

import (
	"errors"
	"strconv"

	"github.com/zadykian/ws/roles/internal/slot"
)

// defaultPort is the UDP port of wg0 where WS_PORT is empty, and forwardPort the TCP port that a
// router forwards to ws's 22 where WS_REMOTE_PORT is.
const (
	defaultPort = 51820
	forwardPort = 22022
)

// Place is where a server sits beside the Mac's others: its slot, and the ports that keep it apart
// from a server behind the same WS_REMOTE.
type Place struct {
	Slot   slot.Slot
	Others []Sibling
}

// FreeSlot returns the smallest slot, 0 to slot.Last, that no other server holds. A server with no
// WS_SLOT line holds slot 0, as the file's parse reads it.
func FreeSlot(others []Sibling) (slot.Slot, error) {
	held := map[slot.Slot]bool{}
	for _, o := range others {
		held[o.Conf.TunnelSlot()] = true
	}
	for s := slot.Slot(0); s <= slot.Last; s++ {
		if !held[s] {
			return s, nil
		}
	}
	return 0, errors.New("every slot, 0 to 15, holds a server: ws client macos remove NAME frees one")
}

// Port is WS_PORT for a server behind remote: 51820 where no other server has the same, non-empty
// WS_REMOTE. Else Port takes a port from 51820 plus the slot up that none of them takes, as the
// router forwards a port to one server.
func (p Place) Port(remote string) string {
	return p.apart(remote, defaultPort, Conf.TunnelPort)
}

// RemotePort is WS_REMOTE_PORT's default behind a router: 22022, or one from 22022 plus the slot up
// likewise.
func (p Place) RemotePort(remote string) string {
	return p.apart(remote, forwardPort, Conf.ForwardPort)
}

// apart returns base where no other server shares remote. Else it returns base plus the slot, or the
// next port up that no server behind remote takes: one placed there alone took base at any slot.
func (p Place) apart(remote string, base int, port func(Conf) string) string {
	taken := map[string]bool{}
	for _, o := range p.Others {
		if remote != "" && o.Conf.Remote == remote {
			taken[port(o.Conf)] = true
		}
	}
	if len(taken) == 0 {
		return strconv.Itoa(base)
	}
	n := base + int(p.Slot)
	for taken[strconv.Itoa(n)] {
		n++
	}
	return strconv.Itoa(n)
}
