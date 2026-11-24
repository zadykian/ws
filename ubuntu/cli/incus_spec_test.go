package cli

import (
	"github.com/zadykian/ws/engine/core/resource/ufw"
	"io"

	"errors"
	enginecli "github.com/zadykian/ws/engine/cli"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
)

const incusPoolCommand = asRoot + selfPath + " ubuntu host incus pool"

// incusUnits loads the host's units for the default values.
func incusUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	units, _ := cableUnits(t, m, "---\n")
	return units
}

func TestIncusPoolRunsWhereItsCheckFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[incusPoolCommand+" --check"] = effecttest.Result{Err: errors.New("to do: create the storage pool ws")}
	m.Results[incusPoolCommand] = effecttest.Result{}
	got := applyUnit(t, incusUnits(t, m), "state/incus-pool")
	if len(got) != 1 || !strings.Contains(got[0], selfPath+" ubuntu host incus pool") {
		t.Errorf("plan %q, want the pool's command", got)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasSuffix(c, incusPoolCommand) && !strings.HasSuffix(c, "--check") }) {
		t.Errorf("calls %q, want the command run as root", m.Calls)
	}
}

func TestIncusPoolIsInSyncWhereItsCheckPasses(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[incusPoolCommand+" --check"] = effecttest.Result{}
	if got := applyUnit(t, incusUnits(t, m), "state/incus-pool"); got != nil {
		t.Errorf("plan %q, want none", got)
	}
}

func TestIncusUnitsRunTheAclThePoolTheBridgeThenTheProfile(t *testing.T) {
	units := incusUnits(t, effecttest.NewMachine())
	var order []string
	for _, u := range units {
		if strings.HasPrefix(u.Name, "state/incus-") && u.Name != "state/incus-ufw" {
			order = append(order, u.Name)
		}
	}
	want := []string{"state/incus-acl", "state/incus-pool", "state/incus-bridge", "state/incus-profile"}
	if !slices.Equal(order, want) {
		t.Errorf("units %q, want %q", order, want)
	}
}

func TestEachIncusUnitRunsItsOwnCommand(t *testing.T) {
	for _, object := range []string{"acl", "bridge", "profile"} {
		command := asRoot + selfPath + " ubuntu host incus " + object
		m := effecttest.NewMachine()
		m.Results[command+" --check"] = effecttest.Result{Err: errors.New("to do")}
		m.Results[command] = effecttest.Result{}
		if got := applyUnit(t, incusUnits(t, m), "state/incus-"+object); len(got) != 1 {
			t.Errorf("%s: plan %q, want the command", object, got)
		}
	}
}

// runIncus runs ws ubuntu host incus ARGS on m, with the host's default values.
func runIncus(m *effecttest.Machine, args ...string) error {
	c := newIncusCmd(enginecli.Env{Machine: m})
	c.SetArgs(args)
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	return c.Execute()
}

func TestIncusCommandsCheckAndMakeTheirObject(t *testing.T) {
	for _, tc := range []struct{ object, list, create string }{
		{"acl", "network acl list", "network acl create ws-egress"},
		{"bridge", "network list", "network create incusbr0 --type bridge"},
		{"profile", "profile list", "profile create ws"},
	} {
		m := effecttest.NewMachine()
		m.Results["incus "+tc.list+" --format json"] = effecttest.Result{Out: []byte("[]")}
		m.Results["incus "+tc.create] = effecttest.Result{}
		if err := runIncus(m, tc.object, "--check"); !errors.Is(err, enginecli.ErrDue) {
			t.Errorf("%s --check = %v, want a change due", tc.object, err)
		}
		if len(m.Inputs) != 0 {
			t.Errorf("%s --check wrote %q", tc.object, m.Inputs)
		}
		if err := runIncus(m, tc.object); err != nil || len(m.Inputs) != 1 {
			t.Errorf("%s = %v, with %d writes, want one create", tc.object, err, len(m.Inputs))
		}
	}
}

func TestIncusPoolWithNoPartitionFailsNamingTheKey(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["incus storage list --format json"] = effecttest.Result{Out: []byte("[]")}
	if err := runIncus(m, "pool"); err == nil || !strings.Contains(err.Error(), "WS_INCUS_POOL_SOURCE") {
		t.Errorf("pool = %v, want the key named", err)
	}
}

func TestIncusUFWLetsDHCPInAndRoutesTheBridge(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[ufw.UserRules] = nil
	want := []string{
		"ufw allow in on incusbr0 from any to any port 67 proto udp",
		"ufw route allow in on incusbr0 from any to any",
	}
	for _, w := range want {
		m.Results[w+" comment incus: the guests' DHCP and routes"] = effecttest.Result{}
	}
	got := applyUnit(t, incusUnits(t, m), "state/incus-ufw")
	if len(got) != len(want) {
		t.Fatalf("plan %q, want the two rules", got)
	}
	for _, w := range want {
		if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, w) }) {
			t.Errorf("calls %q, want %q", m.Calls, w)
		}
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "53") || strings.Contains(c, "out on") {
			t.Errorf("call %q: DNS stays shut, and no rule names an out interface", c)
		}
	}
}
