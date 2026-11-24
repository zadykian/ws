package cli

import (
	"reflect"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// A host holds the DNS rule where a ufw routes, and not the forward policy that a guest needs
// where no ufw does.
func TestHostHoldsTheDNSRuleAndNoForwardPolicy(t *testing.T) {
	host := hostSpecs(t, effecttest.NewMachine())
	if _, ok := host["state/docker-dns"]; !ok {
		t.Error("the host lacks state/docker-dns")
	}
	if _, ok := host["state/docker-forward-policy"]; ok {
		t.Error("the host holds the forward policy of a guest")
	}
}

func TestHostLetsDockersContainersAskTheResolver(t *testing.T) {
	got, err := planned(t, hostSpecs(t, effecttest.NewMachine()), "state/docker-dns")
	want := []string{
		"ufw allow in on docker0 from any to 172.17.0.1 port 53",
		"ufw allow in on br-+ from any to 172.17.0.1 port 53",
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("plan = %q, %v, want %q", got, err, want)
	}
}

// The rule is in before Docker starts, so that no container asks the resolver before it may.
func TestHostLetsTheResolverInBeforeDockerStarts(t *testing.T) {
	names := specLines(t, "host", sideValues(t)["host"])
	rule, start := slices.Index(names, "state/docker-dns"), slices.Index(names, "service/docker")
	if rule < 0 || start < 0 || rule > start {
		t.Errorf("state/docker-dns at %d, service/docker at %d; want the rule first", rule, start)
	}
}
