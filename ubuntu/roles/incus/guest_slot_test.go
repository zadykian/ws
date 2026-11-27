package incus

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

// seedOfSlot1 is the seed that a create at slot 1 wrote.
const seedOfSlot1 = `"cloud-init.network-config":"version: 2\nethernets:\n  eth0:\n    dhcp4: true\n    addresses: [10.99.1.1/32]\n"`

// heldAtSlot1 is heldGuest as a create at slot 1 left it, with no instance named other, which
// the guests then launch.
func heldAtSlot1() string {
	held := strings.Replace(heldGuest, `"name":"other"`, `"name":"foreign"`, 1)
	return strings.Replace(held, `"cloud-init.network-config":"version: 2\n"`, seedOfSlot1, 1)
}

func twoGuests(t *testing.T, labSlot string) []values.Guest {
	other := values.Guest{Name: "other", Address: "10.100.0.11", CPU: 2, Memory: "4GiB", Disk: "100GiB"}
	return []values.Guest{
		resolvedWith(t, lab, "WS_TUNNEL_ADDRESS: "+labSlot+"\n"),
		resolvedWith(t, other, "WS_TUNNEL_ADDRESS: 10.99.3.1/24\n"),
	}
}

func TestGuestsRefuseAHeldGuestWhoseSlotChangedBeforeAnyInstance(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[guestList] = effecttest.Result{Out: []byte(heldAtSlot1())}
	steps, err := Guests(t.Context(), m, twoGuests(t, "10.99.2.1/24"), &effecttest.Out{})
	if err == nil {
		t.Fatal("err = nil, want the refusal of lab's changed slot")
	}
	for _, want := range []string{"lab", "10.99.1.1", "10.99.2.1", "/etc/ws/guests/lab/values.yml", "out of WS_INCUS_GUESTS", "guests remove lab", "old slot still in"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %q, lacks %q", err, want)
		}
	}
	if steps != nil {
		t.Errorf("steps = %v, want none: other, which Incus lacks, must not be launched", steps)
	}
}

func TestGuestsPlanAHeldGuestAtItsOwnSlot(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[guestList] = effecttest.Result{Out: []byte(heldAtSlot1())}
	if _, err := Guests(t.Context(), m, twoGuests(t, "10.99.1.1/24"), &effecttest.Out{}); err != nil {
		t.Fatalf("err = %v, want none: lab keeps slot 1", err)
	}
}

func TestAHeldGuestWhoseRouteDiffersAtItsOwnSlotIsStillEdited(t *testing.T) {
	held := strings.Replace(heldAtSlot1(), `"ipv4.routes":"10.99.1.1/32"`, `"ipv4.routes":"10.99.9.9/32"`, 1)
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[guestList] = effecttest.Result{Out: []byte(held)}
	steps, err := Guests(t.Context(), m, twoGuests(t, "10.99.1.1/24"), &effecttest.Out{})
	if err != nil {
		t.Fatalf("err = %v, want none: the seed names the slot", err)
	}
	changes, err := steps[0].Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
}
