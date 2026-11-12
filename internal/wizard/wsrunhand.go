package wizard

import (
	"cmp"

	"github.com/zadykian/ws/roles/client/macos/conf"
)

// hand adds what stays by hand of ws, the router and the Mac to the list. ws's part comes from the
// remote script's status. ws's gh login is a step of ws server apply, by token.github, so the list
// leaves it out. mac.conf has WS_LAN_ADDRESS with WS_LAN alone, so the lease names an address only
// with a LAN. Off the LAN, away's lines take the router's place.
func (r MacReach) hand(h *Hand, s wsStatus) {
	_, c, err := loadConf(confFile{r.SSH.Conf, runStage, r.SSH.M, r.Out})
	if err != nil {
		return
	}
	fans(h, s)
	if c.Target != "cloud" && s.OutboundOff {
		h.Add("ws's outbound table: off, as ws's /etc/ws/host_vars.yml names security_outbound: false, as for " +
			"a cloud VM; remove that line, then run the wizard with --update (docs/security.md, Outbound).")
	}
	h.Add("macOS's local network: where it asks whether Ghostty, Zed or another app may find " +
		"devices there, allow it (docs/mac.md, ssh's hosts).")
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
		h.Add("ws's lease: %s, as docs/router.md has it, as this run reached ws at %s.", lease, first)
	}
	if c.Remote != "" {
		h.Add("The router's forwards to ws, UDP %s and TCP %s, for %s, as docs/router.md has them.",
			cmp.Or(c.Port, "51820"), cmp.Or(c.RemotePort, "22022"), c.Remote)
	}
}

// hand adds what stays by hand of ws alone, after a run of ws's own wizard. That is the Mac's
// list, less the tunnel, the cable and the Mac's own lines. ws's host_vars names its target, so its
// outbound table follows that, and no mac.conf names another. The wizard adds no device's key to
// ws, so each goes from GitHub by hand.
func (r WsReach) hand(h *Hand, s wsStatus) {
	fans(h, s)
	if c, err := r.Want(); err == nil && c.Target == "lan" {
		h.Add("ws's lease, on a machine on the LAN: static, as docs/router.md has it.")
	}
	h.Add("Each device's ssh key on ws: ssh-import-id gh:LOGIN, run as WS_USER, adds the keys that GitHub " +
		"lists for LOGIN (docs/security.md, SSH and the firewall).")
}

// fans adds the fans' driver where Secure Boot refused it, as the remote script's status read.
func fans(h *Hand, s wsStatus) {
	if s.FansRefused {
		h.Add("ws's fans' driver: with Secure Boot on, DKMS's key, enrolled at ws's console " +
			"(docs/fans.md, Secure Boot).")
	}
}

// away adds what stays by hand of a cloud VM or a VM on the Mac, which have no router. Where the
// run asked ws's address, mac.conf's way must lead to it. A cloud VM's provider must let the tunnel
// in (docs/mac-targets.md).
func away(h *Hand, c conf.Conf, first string) {
	target := c.Target
	way := "WS_LAN_ADDRESS"
	if target == "cloud" {
		way = "WS_REMOTE"
	}
	if first != "" {
		h.Add("ws's address: %s in mac.conf must lead to ws, as this run reached it at %s "+
			"(docs/mac-targets.md).", way, first)
	}
	if target == "cloud" {
		h.Add("The provider's firewall: UDP %s open to ws, for the tunnel (docs/mac-targets.md).",
			cmp.Or(c.Port, "51820"))
	}
}
