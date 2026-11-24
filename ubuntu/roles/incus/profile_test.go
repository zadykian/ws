package incus

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const profileList = "incus profile list --format json"

func TestAMissingProfileIsCreatedUnprivilegedAndIsolated(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[profileList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus profile create ws"] = effecttest.Result{}
	changes, err := Profile(m).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`security.privileged: "false"`, `security.nesting: "true"`, `boot.autostart: "true"`, "snapshots.schedule: '@daily'", "snapshots.expiry: 2w", `security.idmap.isolated: "true"`, "linux.kernel_modules: ",
		"network: incusbr0", `security.mac_filtering: "true"`, `security.port_isolation: "true"`, "pool: ws", "path: /",
	} {
		if !strings.Contains(m.Inputs[0], want) {
			t.Errorf("the profile's YAML lacks %q:\n%s", want, m.Inputs[0])
		}
	}
	if strings.Contains(m.Inputs[0], "br_netfilter") {
		t.Errorf("the profile loads br_netfilter, which a table of the bridge family replaces:\n%s", m.Inputs[0])
	}
}

func TestAProfileThatHoldsItsKeysIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[profileList] = effecttest.Result{Out: []byte(`[{"name":"default","config":{},"devices":{}},
{"name":"ws","description":"` + profileDescription + `","used_by":["/1.0/instances/a"],
"config":{"security.privileged":"false","security.nesting":"true","security.idmap.isolated":"true","linux.kernel_modules":"` + strings.Join(KernelModules, ",") + `","user.note":"x",
"boot.autostart":"true","snapshots.schedule":"@daily","snapshots.expiry":"2w"},
"devices":{"eth0":{"type":"nic","network":"incusbr0","name":"eth0","security.mac_filtering":"true","security.port_isolation":"true"},
"root":{"type":"disk","path":"/","pool":"ws"},"extra":{"type":"none"}}}]`)}
	if changes, err := Profile(m).Plan(context.Background()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none: a key and a device of the host's own stay", changes, err)
	}
}

func TestAProfileThatLostItsIsolationIsEdited(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[profileList] = effecttest.Result{Out: []byte(`[{"name":"ws","description":"` + profileDescription + `",
"config":{"security.privileged":"false","security.nesting":"true","security.idmap.isolated":"false","linux.kernel_modules":"` + strings.Join(KernelModules, ",") + `",
"boot.autostart":"true","snapshots.schedule":"@daily","snapshots.expiry":"2w"},
"devices":{"eth0":{"type":"nic","network":"incusbr0","name":"eth0","security.mac_filtering":"true"},"root":{"type":"disk","path":"/","pool":"ws"}}}]`)}
	changes, err := Profile(m).Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	head, _, _ := strings.Cut(changes[0].String(), "\n")
	if want := "edit the profile ws: config.security.idmap.isolated, devices.eth0.security.port_isolation"; head != want {
		t.Errorf("first line = %q, want %q", head, want)
	}
}
