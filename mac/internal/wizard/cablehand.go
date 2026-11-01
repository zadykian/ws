package wizard

import "context"

// Hand adds the dock to what stays by hand, where mac.conf has no WS_CABLE_SERVICE yet, as no port
// of ws's had carrier. It names WS_CABLE_PORT for the ports that ws keeps down, whose carrier link
// can't read, and an empty WS_CABLE_SERVICE for a Mac without the cable.
func (c *Cable) Hand(_ context.Context, h *Hand) {
	if cf, err := c.Host.conf(); err != nil || cf.Target != "lan" || !c.lacks() {
		return
	}
	text := "The cable: dock the Mac at the display that ws's cable plugs into, then run the line again, " +
		"which takes the cable's steps."
	down := ""
	for _, p := range c.ports {
		if p.carrier == "-" {
			if down != "" {
				down += " or "
			}
			down += p.name
		}
	}
	if down != "" {
		text += " Where the cable plugs into " + down + ", which ws keeps down, run the line with " +
			"WS_CABLE_PORT=PORT in the environment."
	}
	h.Add("%s", text+" Without the cable, run the line once with WS_CABLE_SERVICE= in the environment "+
		"(docs/mac-wizard.md, The cable).")
}
