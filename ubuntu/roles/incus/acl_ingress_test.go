package incus

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/slot"
	"github.com/zadykian/ws/ubuntu/values"
)

func ingressTunnel(name, address string, s slot.Slot, forward ...string) values.GuestTunnel {
	return values.GuestTunnel{
		Guest: values.Guest{Name: name, Address: address}, Slot: s, ForwardFrom: forward,
		Peers: []values.TunnelPeer{{Name: "mac", Address: s.Mac().String()}, {Name: "ipad", Address: s.Server().Next().Next().String()}},
	}
}

func TestEachPeerGetsOneAllowToTheGuestsOwnAddresses(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[aclList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus network acl create ws-ingress"] = effecttest.Result{}
	changes, err := IngressACL(m, []values.GuestTunnel{ingressTunnel("lab", "10.100.0.10", 1)}).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	body := m.Inputs[0]
	for _, want := range []string{"source: 10.99.1.2", "source: 10.99.1.3", "destination: 10.100.0.10,10.99.1.1"} {
		if !strings.Contains(body, want) {
			t.Errorf("the ACL lacks %q:\n%s", want, body)
		}
	}
	if n := strings.Count(body, "action: allow"); n != 2 || strings.Contains(body, "drop") {
		t.Errorf("the ACL has %d allows, want 2 and no drop (the bridge's default drops):\n%s", n, body)
	}
}

func TestAPeerThatTheGuestForwardsForReachesAnyDestination(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[aclList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus network acl create ws-ingress"] = effecttest.Result{}
	tunnel := ingressTunnel("lab", "10.100.0.10", 1, "10.99.1.2")
	changes, err := IngressACL(m, []values.GuestTunnel{tunnel}).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	body := m.Inputs[0]
	if n := strings.Count(body, "destination:"); n != 1 {
		t.Errorf("the ACL names %d destinations, want one: the forwarded-for peer has none:\n%s", n, body)
	}
	if !strings.Contains(body, "source: 10.99.1.2\n") {
		t.Errorf("the ACL lacks the forwarded-for peer:\n%s", body)
	}
}

func TestAnIngressACLWithNoGuestsHoldsNoRule(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[aclList] = effecttest.Result{Out: []byte(`[{"name":"ws-ingress","description":"` + ingressDescription + `","ingress":null,"egress":null,"config":{}}]`)}
	changes, err := IngressACL(m, nil).Plan(context.Background())
	if err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none", changes, err)
	}
}
