package cli

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/ubuntu/roles/incus"
	"github.com/zadykian/ws/ubuntu/values"
)

// runIncusShown is runIncus with what the command prints on its standard error.
func runIncusShown(m *effecttest.Machine, args ...string) (string, error) {
	var shown bytes.Buffer
	c := newIncusCmd(enginecli.Env{Machine: m})
	c.SetArgs(args)
	c.SetOut(io.Discard)
	c.SetErr(&shown)
	err := c.Execute()
	return shown.String(), err
}

func TestIncusGuestsLaunchesWhatTheListDeclaresAndWarnsOfTheRest(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(`[{"name":"old","status":"Running","profiles":["ws"]}]`)}
	m.Files[hostVars] = []byte("WS_INCUS_GUESTS:\n  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}\n")
	m.Files["/etc/ws/guests/lab/values.yml"] = []byte("WS_TUNNEL_ADDRESS: 10.99.1.1/24\n")
	alias, err := incus.Image{M: m}.Alias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	m.Results["incus launch local:"+alias+" lab"] = effecttest.Result{}
	m.Files["/usr/local/bin/ws"] = []byte("binary")
	m.Results["incus file push --create-dirs --uid 0 --gid 0 --mode 0755 /usr/local/bin/ws lab/usr/local/bin/ws"] = effecttest.Result{}
	m.Results["incus file push --create-dirs --uid 0 --gid 0 --mode 0600 /etc/ws/guests/lab/values.yml lab/etc/ws/values.yml"] = effecttest.Result{}
	m.Results["incus exec lab -- sh -c systemctl is-system-running --wait >/dev/null 2>&1; cloud-init status --wait >/dev/null sh"] = effecttest.Result{}
	m.Results["incus exec lab -- /usr/local/bin/ws ubuntu guest setup"] = effecttest.Result{}
	if err := runIncus(m, "guests", "--check"); !errors.Is(err, enginecli.ErrDue) || !strings.Contains(err.Error(), "create the guest lab") {
		t.Errorf("guests --check = %v, want the launch of lab due", err)
	}
	shown, err := runIncusShown(m, "guests")
	if err != nil || len(m.Inputs) == 0 || !strings.Contains(m.Inputs[0], "host_name: ws-lab") {
		t.Errorf("guests = %v, inputs %q, want lab launched", err, m.Inputs)
	}
	if !strings.Contains(shown, "the guest old is not in WS_INCUS_GUESTS: ws deletes none") {
		t.Errorf("stderr %q lacks the warning of the guest old", shown)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "delete") }) {
		t.Errorf("calls %q delete a guest", m.Calls)
	}
	if !slices.Contains(m.Calls, "run incus exec lab -- /usr/local/bin/ws ubuntu guest setup") {
		t.Errorf("calls %q lack the setup of lab", m.Calls)
	}
}

func TestIncusOwnedFillsTheSetFromTheList(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte("WS_INCUS_GUESTS:\n  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}\n")
	m.Results["nft -j list set bridge ws_sources owned4"] = effecttest.Result{Out: []byte(`{"nftables":[{"set":{"name":"owned4"}}]}`)}
	m.Results["nft -f -"] = effecttest.Result{}
	if err := runIncus(m, "owned", "--check"); !errors.Is(err, enginecli.ErrDue) {
		t.Errorf("owned --check = %v, want the element due", err)
	}
	if err := runIncus(m, "owned"); err != nil || len(m.Inputs) != 1 || !strings.Contains(m.Inputs[0], `10.100.0.10 . "ws-lab"`) {
		t.Errorf("owned = %v, inputs %q", err, m.Inputs)
	}
}

// A host that reboots loads the table with no run of ws after it, while Incus starts the guests
// that boot.autostart names. The set must hold their addresses from the load.
func TestSourcesTableLoadsWithTheGuestsAddressesInItsSet(t *testing.T) {
	m, _ := sourcesMachine("chain\n")
	units, _ := cableUnits(t, m, "---\nWS_INCUS_GUESTS:\n"+
		"  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}\n"+
		"  - {name: build, address: 10.100.0.11, cpu: 2, memory: 4GiB, disk: 100GiB}\n")
	applyUnit(t, units, "service/incus-sources")
	want := "elements = { 10.100.0.10 . \"ws-lab\", 10.100.0.11 . \"ws-build\" }"
	if table := string(m.Files[sourcesFile]); !strings.Contains(table, want) {
		t.Errorf("the table lacks %q:\n%s", want, table)
	}
	m, _ = sourcesMachine("chain\n")
	applyUnit(t, incusUnits(t, m), "service/incus-sources")
	if table := string(m.Files[sourcesFile]); strings.Contains(table, "elements") {
		t.Errorf("a host with no guest has elements in its set:\n%s", table)
	}
}

// slotUnits loads the host's specs as a run of ws does, from a host file that names two guests. Each
// guest's own values file gives its slot.
func slotUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	t.Helper()
	m.Files[hostVars] = []byte("WS_INCUS_GUESTS:\n" +
		"  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}\n" +
		"  - {name: build, address: 10.100.0.11, cpu: 2, memory: 4GiB, disk: 100GiB}\n")
	m.Files["/etc/ws/guests/lab/values.yml"] = []byte("WS_TUNNEL_ADDRESS: 10.99.1.1/24\n")
	m.Files["/etc/ws/guests/build/values.yml"] = []byte("WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\n")
	h, err := values.LoadHost(m, hostVars)
	if err != nil {
		t.Fatal(err)
	}
	return loadSpecs(t, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: h, Side: "host", Self: selfPath})
}

// A guest holds its slot's address on eth0, and the peers' replies leave from it. The table drops
// a source of the slots' range that the port does not own.
func TestSourcesTableOwnsAGuestsSlotAddress(t *testing.T) {
	m, _ := sourcesMachine("chain\n")
	applyUnit(t, slotUnits(t, m), "service/incus-sources")
	want := `10.100.0.10 . "ws-lab", 10.99.1.1 . "ws-lab", 10.100.0.11 . "ws-build", 10.99.2.1 . "ws-build"`
	if table := string(m.Files[sourcesFile]); !strings.Contains(table, "elements = { "+want+" }") {
		t.Errorf("the set lacks %q:\n%s", want, table)
	}
}

// What comes in by a guest's port takes its mark before any accept, and only that mark leaves by
// the guest's tunnel interface. So a forged source cannot take another guest's tunnel.
func TestSourcesTableMarksAGuestsPortAndAsksTheMarkOnTheWayOut(t *testing.T) {
	m, _ := sourcesMachine("chain\n")
	applyUnit(t, slotUnits(t, m), "service/incus-sources")
	table := string(m.Files[sourcesFile])
	for _, rule := range []string{
		`iifname "ws-lab" meta mark set 4097`,
		`iifname "ws-build" meta mark set 4098`,
		`oifname "wg1" meta mark != 4097 drop`,
		`oifname "wg2" meta mark != 4098 drop`,
		"table ip ws_guest_out {",
	} {
		if !strings.Contains(table, rule) {
			t.Errorf("the table lacks %q:\n%s", rule, table)
		}
	}
	if set, accept := strings.Index(table, `meta mark set 4097`), strings.Index(table, "@owned4 accept"); set < 0 || accept < set {
		t.Errorf("a guest's mark is set after the first accept:\n%s", table)
	}
}

func TestIncusOwnedFillsTheSetWithAGuestsSlotAddress(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte("WS_INCUS_GUESTS:\n  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}\n")
	m.Files["/etc/ws/guests/lab/values.yml"] = []byte("WS_TUNNEL_ADDRESS: 10.99.1.1/24\n")
	m.Results["nft -j list set bridge ws_sources owned4"] = effecttest.Result{Out: []byte(`{"nftables":[{"set":{"name":"owned4"}}]}`)}
	m.Results["nft -f -"] = effecttest.Result{}
	if err := runIncus(m, "owned"); err != nil || len(m.Inputs) != 1 || !strings.Contains(m.Inputs[0], `10.99.1.1 . "ws-lab"`) {
		t.Errorf("owned = %v, inputs %q, want the slot's address in the set", err, m.Inputs)
	}
}
