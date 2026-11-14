package conf

import (
	"errors"

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
