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

var lab = values.Guest{Name: "lab", Address: "10.100.0.10", CPU: 4, Memory: "8GiB", Disk: "200GiB"}

// heldGuest is what incus lists for a guest ws made. It holds the declared keys and Incus's own,
// such as volatile.* and image.*. A first object has another profile than ws's.
const heldGuest = `[{"name":"other","status":"Running","profiles":["default"]},
{"name":"lab","description":"ws's guest lab","status":"Running","architecture":"x86_64","ephemeral":false,"stateful":false,
"profiles":["ws"],"expanded_config":{"boot.autostart":"true"},
"config":{"limits.cpu":"4","limits.memory":"8GiB","cloud-init.network-config":"version: 2\n","image.os":"Ubuntu","volatile.idmap.base":"1000000","volatile.eth0.hwaddr":"00:16:3e:aa:bb:cc"},
"devices":{"eth0":{"type":"nic","network":"incusbr0","name":"eth0","host_name":"ws-lab","ipv4.address":"10.100.0.10","security.mac_filtering":"true","security.port_isolation":"true"},
"root":{"type":"disk","path":"/","pool":"ws","size":"200GiB"}}}]`

func TestAMissingGuestIsLaunchedWithItsLimitsItsNICAndItsSeed(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus launch local:"+guestAlias+" lab"] = effecttest.Result{}
	changes, err := Guest(m, lab, values.Default(), guestAlias).Plan(t.Context())
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
	if strings.Contains(m.Inputs[0], "10.99.") {
		t.Errorf("the default values name slot 0, which is the host's own, and the seed holds it:\n%s", m.Inputs[0])
	}
}

func TestTheSeedHoldsTheSlotOfAGuestThatNamesOne(t *testing.T) {
	own, err := values.Parse([]byte("WS_TUNNEL_ADDRESS: 10.99.3.1/24\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := "version: 2\nethernets:\n  eth0:\n    dhcp4: true\n    addresses: [10.99.3.1/32]\n"
	if got := seedNetwork(own); got != want {
		t.Errorf("seed = %q, want %q", got, want)
	}
}

func TestAGuestWithItsKeysIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(heldGuest)}
	if changes, err := Guest(m, lab, values.Default(), guestAlias).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none: the seed's key and Incus's own differ from nothing", changes, err)
	}
}

func TestAGuestWithAnotherLimitIsEditedAndKeepsIncusKeys(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(heldGuest)}
	m.Results["incus config edit lab"] = effecttest.Result{}
	more := lab
	more.Memory, more.Disk = "16GiB", "300GiB"
	changes, err := Guest(m, more, values.Default(), guestAlias).Plan(t.Context())
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
	changes, err := Guest(m, moved, values.Default(), guestAlias).Plan(t.Context())
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
	if _, err := Guest(m, lab, values.Default(), guestAlias).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "no guest of ws's") {
		t.Errorf("plan = %v, want the instance refused as no guest of ws's", err)
	}
}
