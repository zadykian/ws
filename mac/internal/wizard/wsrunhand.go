package wizard

import (
	"cmp"
	"context"

	"github.com/zadykian/ws/mac/internal/conf"
)

// Hand adds what stays by hand of ws, the router and the Mac to the list. ws's part comes from the
// remote script's status. ws's gh has a login by apply.sh of WS_HOOK's repository, or else by hand.
// mac.conf has WS_LAN_ADDRESS with WS_LAN alone, so the lease names an address only with a LAN. Off
// the LAN, away's lines take the router's place.
func (r *WsRun) Hand(ctx context.Context, h *Hand) {
	r.Remote.Status(ctx)
	_, c, err := loadConf(confFile{r.Host.Conf, runStage, r.Host.M, r.Host.Out})
	if err != nil {
		return
	}
	if c.Hook == "" {
		h.Add("ws's gh login: gh auth login -h github.com --with-token on ws, with a fine-grained " +
			"token, as mac.conf names no WS_HOOK (README.md, By hand).")
	}
	if r.Remote.Get("fans") == "refused" {
		h.Add("ws's fans' driver: with Secure Boot on, DKMS's key, enrolled at ws's console " +
			"(docs/fans.md, Secure Boot).")
	}
	if c.Target != "cloud" && r.Remote.Get("outbound") == "off" {
		h.Add("ws's outbound table: off, as ws's /etc/ws/host_vars.yml names security_outbound: false, as for " +
			"a cloud VM; remove that line, then run the wizard with --update (docs/security.md, Outbound).")
	}
	h.Add("macOS's local network: where it asks whether Ghostty, Gateway or another app may find " +
		"devices there, allow it (docs/mac.md, ssh's hosts).")
	h.Add("Gateway's connection: an SSH connection to ws in JetBrains Gateway, with Parse config " +
		"file ~/.ssh/config on (docs/mac.md).")
	first := r.Host.SSH.First()
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
