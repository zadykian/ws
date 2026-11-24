package tunnel

import (
	"encoding/hex"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/ufw"
	"github.com/zadykian/ws/ubuntu/values"
)

func TestEachTunnelsRulesTakeACommentOfTheirOwn(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[ufw.UserRules] = []byte("")
	other := labGuestTunnel()
	other.Guest.Name, other.Slot, other.Port = "build", 2, 51822
	got := headlines(t, Firewall{M: m, Tunnels: []values.GuestTunnel{labGuestTunnel(), other}, Bridge: "incusbr0"})
	want := []string{
		"ufw allow in from any to any port 51821 proto udp",
		"ufw route allow in on wg1 out on incusbr0 from any to any",
		"ufw insert 1 deny in on wg1 from any to any",
		"ufw allow in from any to any port 51822 proto udp",
		"ufw route allow in on wg2 out on incusbr0 from any to any",
		"ufw insert 1 deny in on wg2 from any to any",
	}
	if !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
}

func TestTheRulesOfATunnelThatNoGuestHoldsAreDeletedAndWg0sAreKept(t *testing.T) {
	rules := "### tuple ### allow udp 51822 0.0.0.0/0 any 0.0.0.0/0 in comment=" + hexOf("tunnel wg2: its port and its route") + "\n" +
		"### tuple ### allow udp 51820 0.0.0.0/0 any 0.0.0.0/0 in comment=" + hexOf("tunnel wg0: its port") + "\n"
	m := effecttest.NewMachine()
	m.Files[ufw.UserRules] = []byte(rules)
	got := headlines(t, Firewall{M: m, Bridge: "incusbr0"})
	want := []string{"ufw delete allow in from any to any port 51822 proto udp"}
	if !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
}

// The interface of a guest that goes stays, with its key and its peers, and so stays shut to them.
func TestTheDenyOfATunnelThatNoGuestHoldsStays(t *testing.T) {
	rules := "### tuple ### deny any any 0.0.0.0/0 any 0.0.0.0/0 in_wg2 comment=" + hexOf("tunnel wg2: no input") + "\n"
	m := effecttest.NewMachine()
	m.Files[ufw.UserRules] = []byte(rules)
	if got := headlines(t, Firewall{M: m, Bridge: "incusbr0"}); len(got) != 0 {
		t.Errorf("plan = %q, want the deny kept", got)
	}
}

// hexOf is a comment as user.rules writes it.
func hexOf(s string) string { return hex.EncodeToString([]byte(s)) }
