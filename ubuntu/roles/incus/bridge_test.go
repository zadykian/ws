package incus

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

const netList = "incus network list --format json"

var bridgeValues = values.Incus{BridgeAddress: "10.100.0.1/24", BridgeDNS: []string{"9.9.9.9", "1.1.1.1"}}

func TestAMissingBridgeIsCreatedManagedNATedAndWithNoIPv6(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[netList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus network create incusbr0 --type bridge"] = effecttest.Result{}
	changes, err := Bridge(m, bridgeValues).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"ipv4.address: 10.100.0.1/24", `ipv4.nat: "true"`, "ipv6.address: none", "dns.nameservers: 9.9.9.9,1.1.1.1",
		"security.acls: ws-egress", "security.acls.default.egress.action: drop",
	} {
		if !strings.Contains(m.Inputs[0], want) {
			t.Errorf("the bridge's YAML lacks %q:\n%s", want, m.Inputs[0])
		}
	}
}

// oneBridge is what incus lists for a bridge it made, with the keys that it fills.
const oneBridge = `[{"name":"eth0","type":"physical","managed":false,"config":{}},
{"name":"incusbr0","description":"ws's bridge for the guests","type":"bridge","managed":true,"status":"Created","used_by":["/1.0/profiles/ws"],
"config":{"ipv4.address":"10.100.0.1/24","ipv4.nat":"true","ipv6.address":"none","dns.nameservers":"9.9.9.9,1.1.1.1",
"security.acls":"ws-egress","security.acls.default.egress.action":"drop","volatile.network.ipv4.address":"10.100.0.1"}}]`

func TestABridgeThatHoldsItsKeysIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[netList] = effecttest.Result{Out: []byte(oneBridge)}
	if changes, err := Bridge(m, bridgeValues).Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none", changes, err)
	}
}

func TestABridgeOnAnotherSubnetIsEdited(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[netList] = effecttest.Result{Out: []byte(oneBridge)}
	moved := bridgeValues
	moved.BridgeAddress = "10.100.4.1/24"
	changes, err := Bridge(m, moved).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if head, _, _ := strings.Cut(changes[0].String(), "\n"); head != "edit the network incusbr0: config.ipv4.address" {
		t.Errorf("first line = %q", head)
	}
}

func TestANetworkOfTheNameThatIsNoManagedBridgeIsRefused(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[netList] = effecttest.Result{Out: []byte(`[{"name":"incusbr0","type":"bridge","managed":false,"config":{}}]`)}
	if _, err := Bridge(m, bridgeValues).Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "not a managed bridge") {
		t.Errorf("err = %v", err)
	}
}
