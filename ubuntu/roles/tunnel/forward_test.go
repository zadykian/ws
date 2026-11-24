package tunnel

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

const (
	noRules  = `[{"priority":0,"src":"all","table":"local"},{"priority":32766,"src":"all","table":"main"}]`
	labRule  = `{"priority":1001,"src":"all","iif":"wg1","table":"1001"}`
	labRoute = `[{"dst":"default","gateway":"10.100.0.10","dev":"incusbr0","flags":["onlink"]}]`
)

// forwardMachine answers ip with the rules, and the routes of table 1001, that it gets.
func forwardMachine(rules, routes string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["ip -j -4 rule show"] = effecttest.Result{Out: []byte(rules)}
	m.Results["ip -j -4 route show table 1001"] = effecttest.Result{Out: []byte(routes)}
	return m
}

func labGuestTunnel() values.GuestTunnel {
	t := labTunnel()
	t.Guest.Address = "10.100.0.10"
	return t
}

func TestAMissingRuleAndRouteArePlannedRouteFirst(t *testing.T) {
	m := forwardMachine(noRules, "[]")
	got := headlines(t, Forward{M: m, Tunnels: []values.GuestTunnel{labGuestTunnel()}, Bridge: "incusbr0"})
	want := []string{"route table 1001 to lab's address 10.100.0.10 on incusbr0", "send what arrives on wg1 to table 1001, lab's"}
	if !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q: a rule with no route would fall through to the main table", got, want)
	}
}

func TestTheRuleAndTheRouteAreAppliedAsIpTakesThem(t *testing.T) {
	m := forwardMachine(noRules, "[]")
	m.Results["ip -4 route replace default via 10.100.0.10 dev incusbr0 onlink table 1001"] = effecttest.Result{}
	m.Results["ip -4 rule add iif wg1 lookup 1001 priority 1001"] = effecttest.Result{}
	cs, err := Forward{M: m, Tunnels: []values.GuestTunnel{labGuestTunnel()}, Bridge: "incusbr0"}.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestARuleAndARouteAsWantedPlanNothing(t *testing.T) {
	m := forwardMachine("["+labRule+"]", labRoute)
	if got := headlines(t, Forward{M: m, Tunnels: []values.GuestTunnel{labGuestTunnel()}, Bridge: "incusbr0"}); len(got) != 0 {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestARouteToAnotherGuestIsReplacedAndAStaleRuleIsDeleted(t *testing.T) {
	m := forwardMachine(`[`+labRule+`,{"priority":1002,"src":"all","iif":"wg2","table":"1002"},{"priority":1005,"src":"all","iif":"eth0","table":"7"}]`,
		`[{"dst":"default","gateway":"10.100.0.11","dev":"incusbr0"}]`)
	got := headlines(t, Forward{M: m, Tunnels: []values.GuestTunnel{labGuestTunnel()}, Bridge: "incusbr0"})
	want := []string{"route table 1001 to lab's address 10.100.0.10 on incusbr0", "delete the rule of priority 1002 for wg2, which no guest's tunnel wants"}
	if !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q: a rule on eth0 is no tunnel's", got, want)
	}
}

func TestARuleOfAWantedPriorityWithAnotherInterfaceOrTableIsReplaced(t *testing.T) {
	for _, wrong := range []string{
		`{"priority":1001,"src":"all","iif":"wg2","table":"1001"}`,
		`{"priority":1001,"src":"all","iif":"wg1","table":"main"}`,
	} {
		m := forwardMachine("["+wrong+"]", labRoute)
		got := headlines(t, Forward{M: m, Tunnels: []values.GuestTunnel{labGuestTunnel()}, Bridge: "incusbr0"})
		if len(got) != 2 || got[0] != "send what arrives on wg1 to table 1001, lab's" || got[1] != "delete the rule of priority 1001 for "+jsonField(wrong, "iif")+", which no guest's tunnel wants" {
			t.Errorf("rule %s: plan = %q, want the right rule added and the wrong one deleted", wrong, got)
		}
	}
}

// jsonField is the value of a string field of a one-level JSON object, for a test's message.
func jsonField(object, field string) string {
	_, rest, _ := strings.Cut(object, `"`+field+`":"`)
	value, _, _ := strings.Cut(rest, `"`)
	return value
}
