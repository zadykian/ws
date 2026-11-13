package settings

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func peers(p ...values.WireguardPeer) values.Wireguard {
	return values.Wireguard{Address: "10.99.0.1/24", Port: 51820, Peers: p}
}

func TestSettingsPeers(t *testing.T) {
	mac := values.WireguardPeer{Name: "mac", Address: "10.99.0.2"}
	for name, tc := range map[string]struct {
		w    values.Wireguard
		want string
	}{
		"no peers":          {peers(), ""},
		"the Mac":           {peers(mac), ""},
		"two peers":         {peers(mac, values.WireguardPeer{Name: "phone-2", Address: "10.99.0.3"}), ""},
		"upper case":        {peers(values.WireguardPeer{Name: "Mac", Address: "10.99.0.2"}), `A peer's name, "Mac", isn't lowercase letters, digits and dashes.`},
		"an empty name":     {peers(values.WireguardPeer{Address: "10.99.0.2"}), `A peer's name, "", isn't`},
		"a slash":           {peers(values.WireguardPeer{Name: "a/b", Address: "10.99.0.2"}), "isn't lowercase letters"},
		"a name twice":      {peers(mac, values.WireguardPeer{Name: "mac", Address: "10.99.0.3"}), "Two peers are named mac."},
		"no address":        {peers(values.WireguardPeer{Name: "tv"}), "tv's address, , is not an IPv4 address"},
		"IPv6":              {peers(values.WireguardPeer{Name: "tv", Address: "fd00::2"}), "tv's address, fd00::2, is not an IPv4 address"},
		"outside":           {peers(values.WireguardPeer{Name: "tv", Address: "10.98.0.2"}), "tv's address, 10.98.0.2, is not a host's address in 10.99.0.0/24"},
		"the broadcast":     {peers(values.WireguardPeer{Name: "tv", Address: "10.99.0.255"}), "not a host's address in 10.99.0.0/24"},
		"wg0's address":     {peers(values.WireguardPeer{Name: "tv", Address: "10.99.0.1"}), "tv's address, 10.99.0.1, is wg0's too."},
		"another's":         {peers(mac, values.WireguardPeer{Name: "tv", Address: "10.99.0.2"}), "tv's address, 10.99.0.2, is mac's too."},
		"the Mac elsewhere": {peers(values.WireguardPeer{Name: "mac", Address: "10.99.0.5"}), "mac's address is 10.99.0.5, but the Mac's scripts fix it at 10.99.0.2."},
		"the Mac's subnet":  {values.Wireguard{Address: "10.98.0.1/24", Port: 51820, Peers: []values.WireguardPeer{{Name: "mac", Address: "10.99.0.2"}}}, "the Mac's scripts fix wg0 at 10.99.0.1/24"},
		"another subnet":    {values.Wireguard{Address: "10.98.0.1/24", Port: 51820, Peers: []values.WireguardPeer{{Name: "tv", Address: "10.98.0.9"}}}, ""},
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
