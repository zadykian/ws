package incus

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

const (
	guestList  = "incus list --format json"
	guestAlias = "ws-ubuntu-26.04-20261108-0a1b2c3d"
)

// slotted is the tunnel of guest g whose own values hold slot 1.
func slotted(t *testing.T, g values.Guest) values.GuestTunnel {
	own, err := values.Parse([]byte("WS_TUNNEL_ADDRESS: 10.99.1.1/24\n"))
	if err != nil {
		t.Fatal(err)
	}
	tunnel, err := g.Resolve(own).Resolved()
	if err != nil {
		t.Fatal(err)
	}
	return tunnel
}

var lab = values.Guest{Name: "lab", Address: "10.100.0.10", CPU: 4, Memory: "8GiB", Disk: "200GiB"}

// heldGuest is what incus lists for a guest ws made. It holds the declared keys and Incus's own,
// such as volatile.* and image.*. A first object has another profile than ws's.
const heldGuest = `[{"name":"other","status":"Running","profiles":["default"]},
{"name":"lab","description":"ws's guest lab","status":"Running","architecture":"x86_64","ephemeral":false,"stateful":false,
"profiles":["ws"],"expanded_config":{"boot.autostart":"true"},
"config":{"limits.cpu":"4","limits.memory":"8GiB","cloud-init.network-config":"version: 2\n","image.os":"Ubuntu","volatile.idmap.base":"1000000","volatile.eth0.hwaddr":"00:16:3e:aa:bb:cc"},
"devices":{"eth0":{"type":"nic","network":"incusbr0","name":"eth0","host_name":"ws-lab","ipv4.address":"10.100.0.10","ipv4.routes":"10.99.1.1/32","security.mac_filtering":"true","security.port_isolation":"true"},
"root":{"type":"disk","path":"/","pool":"ws","size":"200GiB"}}}]`

func TestAMissingGuestIsLaunchedWithItsLimitsItsNICAndItsSeed(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus launch local:"+guestAlias+" lab"] = effecttest.Result{}
	changes, err := Guest(m, slotted(t, lab), false, guestAlias).Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if head, _, _ := strings.Cut(changes[0].String(), "\n"); head != "create the guest lab" {
		t.Errorf("first line = %q", head)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"profiles:\n    - ws\n", `limits.cpu: "4"`, "limits.memory: 8GiB", "host_name: ws-lab", "ipv4.address: 10.100.0.10",
		"network: incusbr0", "size: 200GiB", "pool: ws", "cloud-init.network-config: |\n", "dhcp4: true",
	} {
		if !strings.Contains(m.Inputs[0], want) {
			t.Errorf("the guest's YAML lacks %q:\n%s", want, m.Inputs[0])
		}
	}
	if !strings.Contains(m.Inputs[0], "addresses: [10.99.1.1/32]") {
		t.Errorf("the seed lacks the slot's address:\n%s", m.Inputs[0])
	}
}

func TestAGuestWithItsKeysIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(heldGuest)}
	if changes, err := Guest(m, slotted(t, lab), false, guestAlias).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none: the seed's key and Incus's own differ from nothing", changes, err)
	}
}

func TestAGuestWithAnotherLimitIsEditedAndKeepsIncusKeys(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(heldGuest)}
	m.Results["incus config edit lab"] = effecttest.Result{}
	more := lab
	more.Memory, more.Disk = "16GiB", "300GiB"
	changes, err := Guest(m, slotted(t, more), false, guestAlias).Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if head, _, _ := strings.Cut(changes[0].String(), "\n"); head != "edit the guest lab: config.limits.memory, devices.root.size" {
		t.Errorf("first line = %q", head)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"limits.memory: 16GiB", "size: 300GiB", "volatile.idmap.base", "image.os: Ubuntu", "architecture: x86_64", "volatile.eth0.hwaddr"} {
		if !strings.Contains(m.Inputs[0], want) {
			t.Errorf("the edit's YAML lacks %q:\n%s", want, m.Inputs[0])
		}
	}
	if strings.Contains(m.Inputs[0], "expanded") || strings.Contains(m.Inputs[0], "status") {
		t.Errorf("the edit sends keys that are no instance's to write:\n%s", m.Inputs[0])
	}
}

func TestAGuestWithAnotherAddressOrPortIsEdited(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(heldGuest)}
	moved := lab
	moved.Address = "10.100.0.11"
	changes, err := Guest(m, slotted(t, moved), false, guestAlias).Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if head, _, _ := strings.Cut(changes[0].String(), "\n"); head != "edit the guest lab: devices.eth0.ipv4.address" {
		t.Errorf("first line = %q", head)
	}
}

func TestAnInstanceOfAnotherProfileIsNoGuestOfWs(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(`[{"name":"lab","profiles":["default"]}]`)}
	if _, err := Guest(m, slotted(t, lab), false, guestAlias).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "no guest of ws's") {
		t.Errorf("plan = %v, want the instance refused as no guest of ws's", err)
	}
}

func TestAGuestWithTheSwitchOnLaunchesWithNVIDIAsCardAndWithoutItNone(t *testing.T) {
	for own, want := range map[bool]bool{true: true, false: false} {
		m := effecttest.NewMachine()
		m.Results[guestList] = effecttest.Result{Out: []byte("[]")}
		m.Results["incus launch local:"+guestAlias+" lab"] = effecttest.Result{}
		changes, err := Guest(m, slotted(t, lab), own, guestAlias).Plan(t.Context())
		if err != nil || len(changes) != 1 {
			t.Fatalf("switch %t: changes = %v, err = %v", own, changes, err)
		}
		if err := changes[0].Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(m.Inputs[0], "type: gpu") && strings.Contains(m.Inputs[0], "vendorid: 10de"); got != want {
			t.Errorf("switch %t: the guest's YAML holds the card = %t, want %t:\n%s", own, got, want, m.Inputs[0])
		}
		if strings.Contains(m.Inputs[0], "null") {
			t.Errorf("switch %t: the YAML names a key as null:\n%s", own, m.Inputs[0])
		}
	}
}

func TestAGuestWhoseSwitchTurnsOnGetsTheCardAndKeepsItsOtherDevices(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(heldGuest)}
	m.Results["incus config edit lab"] = effecttest.Result{}
	changes, err := Guest(m, slotted(t, lab), true, guestAlias).Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if head, _, _ := strings.Cut(changes[0].String(), "\n"); head != "edit the guest lab: devices.gpu" {
		t.Errorf("first line = %q", head)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"gpu:", "vendorid: 10de", "host_name: ws-lab", "volatile.idmap.base"} {
		if !strings.Contains(m.Inputs[0], want) {
			t.Errorf("the edit's YAML lacks %q:\n%s", want, m.Inputs[0])
		}
	}
}

func TestAGuestWhoseSwitchTurnsOffLosesTheCard(t *testing.T) {
	held := strings.Replace(heldGuest, `"root":{"type":"disk"`, `"gpu":{"type":"gpu","vendorid":"10de"},"root":{"type":"disk"`, 1)
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(held)}
	m.Results["incus config edit lab"] = effecttest.Result{}
	if changes, err := Guest(m, slotted(t, lab), true, guestAlias).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Fatalf("with the switch on: changes = %v, err = %v, want none", changes, err)
	}
	changes, err := Guest(m, slotted(t, lab), false, guestAlias).Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if head, _, _ := strings.Cut(changes[0].String(), "\n"); head != "edit the guest lab: devices.gpu" {
		t.Errorf("first line = %q", head)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.Inputs[0], "gpu") || !strings.Contains(m.Inputs[0], "host_name: ws-lab") {
		t.Errorf("the edit's YAML keeps the card or loses the NIC:\n%s", m.Inputs[0])
	}
}

func TestAGuestOfASlotGetsTheSlotsAddressAsARouteOnItsNIC(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus launch local:"+guestAlias+" lab"] = effecttest.Result{}
	changes, err := Guest(m, slotted(t, lab), false, guestAlias).Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.Inputs[0], "ipv4.routes: 10.99.1.1/32") {
		t.Errorf("the NIC lacks the slot's route:\n%s", m.Inputs[0])
	}
}

// resolvedWith is guest g resolved from the own values that text gives, as the host's load does.
func resolvedWith(t *testing.T, g values.Guest, text string) values.Guest {
	own, err := values.Parse([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return g.Resolve(own)
}

func TestGuestsRefuseAnAddressOfAnotherPrefixLengthBeforeAnyInstance(t *testing.T) {
	m := effecttest.NewMachine()
	guest := resolvedWith(t, lab, "WS_TUNNEL_ADDRESS: 10.99.3.1/25\nWS_TUNNEL_PEERS: []\n")
	_, err := Guests(t.Context(), m, []values.Guest{guest}, &effecttest.Out{})
	if err == nil || !strings.Contains(err.Error(), "not a slot's 10.99.N.1/24") {
		t.Fatalf("err = %v, want the tunnel's refusal of the /25", err)
	}
	if len(m.Calls) != 0 {
		t.Errorf("calls = %q, want none: the plan fails before it asks Incus", m.Calls)
	}
}

func TestGuestsRefuseAGuestOfSlotZeroAmongSeveral(t *testing.T) {
	m := effecttest.NewMachine()
	other := values.Guest{Name: "other", Address: "10.100.0.11", CPU: 2, Memory: "4GiB", Disk: "100GiB"}
	guests := []values.Guest{resolvedWith(t, lab, "WS_TUNNEL_ADDRESS: 10.99.1.1/24\n"), resolvedWith(t, other, "")}
	_, err := Guests(t.Context(), m, guests, &effecttest.Out{})
	if err == nil || !strings.Contains(err.Error(), "other: WS_TUNNEL_ADDRESS is 10.99.0.1/24, slot 0") {
		t.Fatalf("err = %v, want the refusal of other's slot 0", err)
	}
	if len(m.Calls) != 0 {
		t.Errorf("calls = %q, want none", m.Calls)
	}
}

func TestGuestsMakeTheInstanceOfAGuestOfASlotFromItsTunnel(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[guestList] = effecttest.Result{Out: []byte("[]")}
	guest := resolvedWith(t, lab, "WS_TUNNEL_ADDRESS: 10.99.3.1/24\n")
	alias, err := Image{M: m}.Alias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	m.Results["incus launch local:"+alias+" lab"] = effecttest.Result{}
	steps, err := Guests(t.Context(), m, []values.Guest{guest}, &effecttest.Out{})
	if err != nil || len(steps) != 5 {
		t.Fatalf("steps = %d, err = %v", len(steps), err)
	}
	changes, err := steps[0].Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ipv4.routes: 10.99.3.1/32", "addresses: [10.99.3.1/32]"} {
		if !strings.Contains(m.Inputs[0], want) {
			t.Errorf("the guest's YAML lacks %q:\n%s", want, m.Inputs[0])
		}
	}
}

// A guest that no load resolved has no tunnel to give, and the plan fails before it asks Incus.
func TestGuestsRefuseAGuestThatNoLoadResolved(t *testing.T) {
	m := effecttest.NewMachine()
	_, err := Guests(t.Context(), m, []values.Guest{lab}, &effecttest.Out{})
	if err == nil || !strings.Contains(err.Error(), "not loaded") || len(m.Calls) != 0 {
		t.Errorf("err = %v, calls = %q, want the guest named as not loaded and no call", err, m.Calls)
	}
}

// A guest's GPU switch is its own values': Guests reads it off the resolved guest.
func TestGuestsGiveTheGPUToAGuestWhoseValuesSetIt(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[guestList] = effecttest.Result{Out: []byte("[]")}
	guest := resolvedWith(t, lab, "WS_TUNNEL_ADDRESS: 10.99.3.1/24\nWS_NVIDIA_GPU: true\n")
	alias, err := Image{M: m}.Alias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	m.Results["incus launch local:"+alias+" lab"] = effecttest.Result{}
	steps, err := Guests(t.Context(), m, []values.Guest{guest}, &effecttest.Out{})
	if err != nil {
		t.Fatal(err)
	}
	changes, err := steps[0].Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.Inputs[0], "vendorid") {
		t.Errorf("the guest's YAML lacks the gpu device:\n%s", m.Inputs[0])
	}
}
