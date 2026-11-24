package kind

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/ufw"
	"github.com/zadykian/ws/engine/core/step"
)

func TestUFWOutIsARouteRule(t *testing.T) {
	e, m, _, _ := testEnv()
	// The route rule from 10.99.0.3 carries the comment ikeward, 696b6577617264 in hex.
	m.Files[ufw.UserRules] = []byte("### tuple ### route:allow any any 0.0.0.0/0 any 10.99.0.3 in_wg0!out_ikeward0 comment=696b6577617264\n")
	units := loadState(t, e, "ufw = { comment = \"ikeward\", rules = [{ action = \"allow\", interface = \"wg0\", out = \"ikeward0\", from = \"10.99.0.2\" }] }\n")
	got := headlines(t, units[0].Step)
	p, err := ufw.Rules{M: m, Comment: "ikeward", Want: []ufw.Rule{ufw.Route("allow", "wg0", "ikeward0", "10.99.0.2", "ikeward")}}.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, c := range p {
		want = append(want, step.Headline(c))
	}
	if len(want) != 2 || !slices.Equal(got, want) {
		t.Errorf("plan = %q, want ufw.Route's %q", got, want)
	}
}

func TestUFWOutRefusesPortsAndDir(t *testing.T) {
	e, _, _, _ := testEnv()
	for _, extra := range []string{`proto = "udp"`, `to = "10.0.0.1"`, `to_port = "53"`, `from_port = "53"`, `dir = "in"`} {
		stateRefused(t, e, "ufw = { comment = \"c\", rules = [{ action = \"allow\", interface = \"wg0\", out = \"x0\", "+extra+" }] }\n",
			"takes action, interface, out and from alone")
	}
	stateRefused(t, e, "ufw = { comment = \"c\", rules = [{ action = \"allow\", out = \"x0\" }] }\n", "takes interface")
}

func TestUFWOutAnyRoutesToEveryInterface(t *testing.T) {
	e, m, _, _ := testEnv()
	// The route rule from every address carries the comment guests, 677565737473 in hex.
	m.Files[ufw.UserRules] = []byte("### tuple ### route:allow any any 0.0.0.0/0 any 0.0.0.0/0 in_br0 comment=677565737473\n")
	units := loadState(t, e, "ufw = { comment = \"guests\", rules = [{ action = \"allow\", interface = \"br0\", out = \"any\" }] }\n")
	if got := headlines(t, units[0].Step); len(got) != 0 {
		t.Errorf("plan = %q, want none: the rule is in force", got)
	}
	m.Files[ufw.UserRules] = nil
	if got := headlines(t, units[0].Step); len(got) != 1 {
		t.Errorf("plan = %q, want the rule added", got)
	}
}
