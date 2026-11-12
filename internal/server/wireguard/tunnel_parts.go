package wireguard

import (
	"context"
	"io"
	"strconv"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/ufw"
	"github.com/zadykian/ws/internal/server/values"
)

// ufwComment marks the rule the role owns.
const ufwComment = "WireGuard"

// firewall lets in UDP on the port on every interface, as the Mac reaches the port over the
// cable, the LAN and the router's forward. Inside the tunnel, ufw takes wg0 as any interface: it
// lets in SSH, and forwards only ping and what the ikeward role lets through to the VPN. Docker
// forwards ahead of it. A rule for a port that wireguard_port named before is removed; one made by
// hand, without the comment, stays.
func firewall(m effect.Machine, out effect.Out, port int) ufw.Rules {
	rule := ufw.Rule{Action: "allow", Proto: "udp", To: ufw.Any, ToPort: strconv.Itoa(port),
		From: ufw.Any, FromPort: "any", Dir: "in"}
	return ufw.Rules{M: m, Out: out, Comment: ufwComment, Want: []ufw.Rule{rule}, NeedsSystemd: true,
		Owns: func(r ufw.Rule) bool {
			return r.Action == "allow" && r.Proto == "udp" && r.To == ufw.Any && r.FromPort == "any" &&
				r.Dir == "in" && r.Interface == ""
		}}
}

// tunnel makes the resources of wg0: the check of networkd, the key, the .netdev and the .network
// files, the firewall, and the reload of networkd. The files are for networkd alone, so they are
// the key's group and mode, as systemd.netdev(5) suggests for a WireGuard .netdev. Their
// diff holds public keys and addresses alone.
func tunnel(m effect.Machine, u effect.Units, out effect.Out, random io.Reader, v values.Values) func(context.Context) ([]host.Resource, error) {
	return func(context.Context) ([]host.Resource, error) {
		gid, err := groupID(m)
		if err != nil {
			return nil, err
		}
		peers, err := peerKeys(m, out, v.Wireguard)
		if err != nil {
			return nil, err
		}
		netdev, err := Netdev(v.Wireguard.Port, peers)
		if err != nil {
			return nil, err
		}
		network, err := Network(v.Wireguard)
		if err != nil {
			return nil, err
		}
		k := key{m: m, random: random, gid: gid}
		netdevRes := host.File{M: m, Path: netdevFile, Content: netdev, Mode: 0o640, GID: gid}
		networkRes := host.File{M: m, Path: networkFile, Content: network, Mode: 0o644}
		return []host.Resource{
			networkdCheck{m, u}, k, netdevRes, networkRes, firewall(m, out, v.Wireguard.Port),
			wg0{m: m, watch: []host.Resource{k, netdevRes, networkRes}},
		}, nil
	}
}
