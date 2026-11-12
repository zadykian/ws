package security

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/ufw"
)

// sshComment marks the rule that is ws's.
const sshComment = "ws: SSH"

// firewall lets SSH in, then denies incoming connections and allows outgoing ones by default, then
// enables ufw. ufw needs the kernel's netfilter even to show its state, so each of these needs a
// booted machine. SSH's rule goes in first, so that no default policy refuses SSH, not even to a
// run over SSH, whatever state ufw starts in. The outbound table holds what goes out where. ufw
// can't match resolved's or networkd's uid, or an IPv6 next hop. Docker forwards its containers'
// packets ahead of ufw's rules. ufw.service loads the rules at boot; without it, the machine would
// boot with no firewall.
func firewall(m effect.Machine, u effect.Units, out effect.Out) []host.Resource {
	ssh := ufw.Rule{Action: "allow", Proto: "tcp", To: "any", ToPort: "22", From: "any", FromPort: "any", Dir: "in"}
	return []host.Resource{
		// ufw writes a rule for IPv6 too, which is not in Want and which Owns leaves alone.
		ufw.Rules{M: m, Out: out, Comment: sshComment, Want: []ufw.Rule{ssh}, NeedsSystemd: true,
			Owns: func(r ufw.Rule) bool { return r.To == ufw.Any }},
		ufw.Defaults{M: m, Out: out, Incoming: "deny", Outgoing: "allow"},
		ufw.Enable{M: m, Out: out},
		host.Unit{M: m, U: u, Out: out, Name: "ufw.service", Enabled: true},
	}
}
