package settings

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func peers(p ...values.TunnelPeer) values.Tunnel {
	return values.Tunnel{Address: "10.99.0.1/24", Port: 51820, Peers: p}
}

func slotMac(address, mac string) values.Tunnel {
	return values.Tunnel{Address: address, Port: 51820, Peers: []values.TunnelPeer{{Name: "mac", Address: mac}}}
}

func TestSettingsPeers(t *testing.T) {
	mac := values.TunnelPeer{Name: "mac", Address: "10.99.0.2"}
	for name, tc := range map[string]struct {
		w    values.Tunnel
		want string
	}{
		"no peers":          {peers(), ""},
		"the Mac":           {peers(mac), ""},
		"two peers":         {peers(mac, values.TunnelPeer{Name: "phone-2", Address: "10.99.0.3"}), ""},
		"upper case":        {peers(values.TunnelPeer{Name: "Mac", Address: "10.99.0.2"}), `A peer's name, "Mac", isn't lowercase letters, digits and dashes.`},
		"an empty name":     {peers(values.TunnelPeer{Address: "10.99.0.2"}), `A peer's name, "", isn't`},
		"a slash":           {peers(values.TunnelPeer{Name: "a/b", Address: "10.99.0.2"}), "isn't lowercase letters"},
		"a name twice":      {peers(mac, values.TunnelPeer{Name: "mac", Address: "10.99.0.3"}), "Two peers are named mac."},
		"no address":        {peers(values.TunnelPeer{Name: "tv"}), "tv's address, , is not an IPv4 address"},
		"IPv6":              {peers(values.TunnelPeer{Name: "tv", Address: "fd00::2"}), "tv's address, fd00::2, is not an IPv4 address"},
		"outside":           {peers(values.TunnelPeer{Name: "tv", Address: "10.98.0.2"}), "tv's address, 10.98.0.2, is not a host's address in 10.99.0.0/24"},
		"the broadcast":     {peers(values.TunnelPeer{Name: "tv", Address: "10.99.0.255"}), "not a host's address in 10.99.0.0/24"},
		"wg0's address":     {peers(values.TunnelPeer{Name: "tv", Address: "10.99.0.1"}), "tv's address, 10.99.0.1, is wg0's too."},
		"another's":         {peers(mac, values.TunnelPeer{Name: "tv", Address: "10.99.0.2"}), "tv's address, 10.99.0.2, is mac's too."},
		"the Mac elsewhere": {peers(values.TunnelPeer{Name: "mac", Address: "10.99.0.5"}), "mac's address is 10.99.0.5, but the Mac's scripts fix it at 10.99.0.2."},
		"the Mac's subnet":  {values.Tunnel{Address: "10.98.0.1/24", Port: 51820, Peers: []values.TunnelPeer{{Name: "mac", Address: "10.99.0.2"}}}, "the Mac's scripts fix wg0 at a slot's 10.99.N.1/24, N from 0 to 15."},
		"slot 1":            {slotMac("10.99.1.1/24", "10.99.1.2"), ""},
		"the last slot":     {slotMac("10.99.15.1/24", "10.99.15.2"), ""},
		"mac at slot 0":     {slotMac("10.99.1.1/24", "10.99.0.2"), "fix it at 10.99.1.2"},
		"past the slots":    {slotMac("10.99.16.1/24", "10.99.16.2"), "slot's 10.99.N.1/24"},
		"a wide subnet":     {slotMac("10.99.1.1/20", "10.99.1.2"), "slot's 10.99.N.1/24"},
		"another subnet":    {values.Tunnel{Address: "10.98.0.1/24", Port: 51820, Peers: []values.TunnelPeer{{Name: "tv", Address: "10.98.0.9"}}}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			err := Validate(tc.w)
			switch {
			case tc.want == "" && err != nil:
				t.Errorf("Validate = %v, want none", err)
			case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
				t.Errorf("Validate = %v, want a refusal with %q", err, tc.want)
			}
		})
	}
}
