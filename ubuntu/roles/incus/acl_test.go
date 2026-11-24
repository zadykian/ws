package incus

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

const aclList = "incus network acl list --format json"

func TestAMissingEgressACLDropsThePrivateRangesAndAllowsTheRest(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[aclList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus network acl create ws-egress"] = effecttest.Result{}
	changes, err := EgressACL(m).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	body := m.Inputs[0]
	// The ranges of the old outbound table, with the host's own addresses among them.
	for _, r := range []string{"10.0.0.0/8", "100.64.0.0/10", "169.254.0.0/16", "172.16.0.0/12", "192.168.0.0/16", "224.0.0.0/4", "255.255.255.255/32"} {
		if !strings.Contains(body, r) {
			t.Errorf("the ACL lacks %s:\n%s", r, body)
		}
	}
	drop, allow := strings.Index(body, "action: drop"), strings.Index(body, "action: allow")
	if drop < 0 || allow < 0 || strings.Contains(body, "ingress") {
		t.Errorf("the ACL has a drop and an allow, and no ingress, which ws-ingress holds:\n%s", body)
	}
}

func TestAnEgressACLWithItsRulesIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[aclList] = effecttest.Result{Out: []byte(`[{"name":"ws-egress","description":"` + aclDescription + `","used_by":["/1.0/networks/incusbr0"],
"ingress":null,"egress":[{"action":"drop","state":"enabled","destination":"` + strings.Join(values.PrivateRanges, ",") + `","description":"` + dropDescription + `"},
{"action":"allow","state":"enabled","description":"` + allowDescription + `"}],"config":{}}]`)}
	changes, err := EgressACL(m).Plan(context.Background())
	if err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none", changes, err)
	}
}

func TestAnEgressACLWithAnAllowedRangeIsEdited(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[aclList] = effecttest.Result{Out: []byte(`[{"name":"ws-egress","egress":[{"action":"allow","state":"enabled"}]}]`)}
	changes, err := EgressACL(m).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if head, _, _ := strings.Cut(changes[0].String(), "\n"); head != "edit the network ACL ws-egress: description, egress" {
		t.Errorf("first line = %q", head)
	}
}
