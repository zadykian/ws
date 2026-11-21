package kind

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/ufw"
)

func TestUFWOwnsLeavesARuleOfAnotherShape(t *testing.T) {
	e, m, _, _ := testEnv()
	// Each rule carries the comment WG, 5747 in hex: the old port's, and two made by hand on wg0.
	m.Files[ufw.UserRules] = []byte("### tuple ### allow udp 51821 0.0.0.0/0 any 0.0.0.0/0 in comment=5747\n" +
		"### tuple ### allow tcp 8000 0.0.0.0/0 any 0.0.0.0/0 in_wg0 comment=5747\n" +
		"### tuple ### allow udp 9000 0.0.0.0/0 any 0.0.0.0/0 in_wg0 comment=5747\n")
	units := loadState(t, e, "ufw = { comment = \"WG\", owns = { action = \"allow\", proto = \"udp\", to = \"any\", "+
		"from_port = \"any\", dir = \"in\", interface = \"\" }, rules = [{ action = \"allow\", proto = \"udp\", to_port = \"51820\", dir = \"in\" }] }\n")
	want := []string{"ufw allow in from any to any port 51820 proto udp", "ufw delete allow in from any to any port 51821 proto udp"}
	if got := headlines(t, units[0].Step); !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
	stateRefused(t, e, "ufw = { comment = \"c\", owns = { port = \"53\" }, rules = [] }\n", `owns: "port" is no field of a rule`)
}
