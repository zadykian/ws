package host

import (
	"cmp"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"

	"github.com/zadykian/ws/roles/client/macos/conf"
)

// Hand adds what stays by hand of ws, the router and the Mac to the list. ws's part comes from the
// remote script's status. ws's gh login is a step of ws server ubuntu apply, by token.github, so
// the list leaves it out. mac.conf has WS_LAN_ADDRESS with WS_LAN alone, so the lease names an
// address only with a LAN. Off the LAN, away's lines take the router's place.
func (r MacReach) Hand(h *guided.Hand, s session.Status) {
	_, c, err := (conf.File{Path: r.Conf, Stage: stages.RunStage, M: r.SSH.M, Out: r.Out}).Load()
	if err != nil {
		return
	}
	session.Fans(h, s)
	if c.Target != "cloud" && s.OutboundOff {
		h.Add("ws's outbound table: off, as ws's /etc/ws/host_vars.yml names security_outbound: false, as for " +
			"a cloud VM; remove that line, then run the wizard with --update (roles/server/ubuntu/security/README.md, Outbound).")
	}
	h.Add("macOS's local network: where it asks whether Ghostty, Zed or another app may find " +
		"devices there, allow it (roles/client/macos/README.md, ssh's hosts).")
	first := r.SSH.First()
	if c.Target != "lan" {
		away(h, c, first)
		return
	}
	if first != "" {
		lease := "static"
		if c.LANAddress != "" {
			lease += " at " + c.LANAddress
		}
		h.Add("ws's lease: %s, as roles/server/ubuntu/network/router.md has it, as this run reached ws at %s.", lease, first)
	}
	if c.Remote != "" {
		h.Add("The router's forwards to ws, UDP %s and TCP %s, for %s, as roles/server/ubuntu/network/router.md has them.",
			cmp.Or(c.Port, "51820"), cmp.Or(c.RemotePort, "22022"), c.Remote)
	}
}

// away adds what stays by hand of a cloud VM or a VM on the Mac, which have no router. Where the
// run asked ws's address, mac.conf's way must lead to it. A cloud VM's provider must let the tunnel
// in (roles/client/macos/conf/README.md).
func away(h *guided.Hand, c conf.Conf, first string) {
	target := c.Target
	way := "WS_LAN_ADDRESS"
	if target == "cloud" {
		way = "WS_REMOTE"
	}
	if first != "" {
		h.Add("ws's address: %s in mac.conf must lead to ws, as this run reached it at %s "+
			"(roles/client/macos/conf/README.md).", way, first)
	}
	if target == "cloud" {
		h.Add("The provider's firewall: UDP %s open to ws, for the tunnel (roles/client/macos/conf/README.md).",
			cmp.Or(c.Port, "51820"))
	}
}
