package ufw

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestCallsRunUfwAsRootWithTheComment(t *testing.T) {
	r := Rule{Action: "allow", Proto: "any", ToPort: "53", To: "172.17.0.1", FromPort: "any",
		From: Any, Dir: "in", Interface: "docker0", Comment: "DNS from containers"}
	m := effecttest.NewMachine()
	add := "ufw --force allow in on docker0 from any to 172.17.0.1 port 53 comment DNS from containers"
	del := "ufw --force delete allow in on docker0 from any to 172.17.0.1 port 53 comment DNS from containers"
	m.Results[add], m.Results[del] = effecttest.Result{}, effecttest.Result{}
	for _, c := range []ufwCall{{m, false, r}, {m, true, r}} {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"root " + add, "root " + del}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

func TestCallsDeleteARouteWithRouteFirst(t *testing.T) {
	r := Route("allow", "wg0", "ikeward0", "10.99.0.2", "ikeward")
	m := effecttest.NewMachine()
	del := "ufw --force route delete allow in on wg0 out on ikeward0 from 10.99.0.2 to any comment ikeward"
	m.Results[del] = effecttest.Result{}
	if err := (ufwCall{m, true, r}).Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"root " + del}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}
