package cable

import "github.com/zadykian/ws/engine/core/effect"

// direct is the cable from the Mac's dock to the server: the server's end gets Address from a
// netplan file of its own. With Interface empty, the file goes, and so does the address. A machine
// without the cable's port, as a cloud VM or CI's container, keeps its netplan files as they are.
type direct struct {
	M                  effect.Machine
	U                  effect.Units
	Out                effect.Out
	Interface, Address string
}
