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
	"github.com/zadykian/ws/ubuntu/roles/incus"
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
	alias, err := incus.Image{M: m}.Alias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	m.Results["incus launch local:"+alias+" lab"] = effecttest.Result{}
	if err := runIncus(m, "guests", "--check"); !errors.Is(err, enginecli.ErrDue) || !strings.Contains(err.Error(), "create the guest lab") {
		t.Errorf("guests --check = %v, want the launch of lab due", err)
	}
	shown, err := runIncusShown(m, "guests")
	if err != nil || len(m.Inputs) != 1 || !strings.Contains(m.Inputs[0], "host_name: ws-lab") {
		t.Errorf("guests = %v, inputs %q, want lab launched", err, m.Inputs)
	}
	if !strings.Contains(shown, "the guest old is not in WS_INCUS_GUESTS: ws deletes none") {
		t.Errorf("stderr %q lacks the warning of the guest old", shown)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "delete") }) {
		t.Errorf("calls %q delete a guest", m.Calls)
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
