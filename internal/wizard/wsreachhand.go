package wizard

import guided "github.com/zadykian/ws/engine/core/wizard"

// hand adds what stays by hand of ws alone, after a run of ws's own wizard. That is the Mac's
// list, less the tunnel, the cable and the Mac's own lines. ws's host_vars names its target, so its
// outbound table follows that, and no mac.conf names another. The wizard adds no device's key to
// ws, so each goes from GitHub by hand.
func (r WsReach) hand(h *guided.Hand, s wsStatus) {
	fans(h, s)
	if c, err := r.Want(); err == nil && c.Target == "lan" {
		h.Add("ws's lease, on a machine on the LAN: static, as docs/router.md has it.")
	}
	h.Add("Each device's ssh key on ws: ssh-import-id gh:LOGIN, run as WS_USER, adds the keys that GitHub " +
		"lists for LOGIN (docs/security.md, SSH and the firewall).")
}

// fans adds the fans' driver where Secure Boot refused it, as the remote script's status read.
func fans(h *guided.Hand, s wsStatus) {
	if s.FansRefused {
		h.Add("ws's fans' driver: with Secure Boot on, DKMS's key, enrolled at ws's console " +
			"(docs/fans.md, Secure Boot).")
	}
}
