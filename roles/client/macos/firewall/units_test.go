package firewall_test

import (
	"os"
	"os/exec"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/facts"
)

// The run state's lines: its check, which runs as the user, and its command, which runs as root.
const (
	socketfilterfw = "/usr/libexec/ApplicationFirewall/socketfilterfw"
	inPath         = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
	script         = socketfilterfw + " --getglobalstate | grep -Eqi 'enabled|block all' && " +
		socketfilterfw + " --getblockall | grep -qi 'enabled'"
	check  = inPath + "sh -c " + script
	enable = inPath + socketfilterfw + " --setglobalstate on --setblockall on"
)

// macUnits loads the role's units on a Mac of the user v.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, System: kind.MacOS, User: "v",
		Data: facts.Mac{User: "v", Home: "/Users/v"}}, fstest.MapFS{"firewall/units.toml.tmpl": &fstest.MapFile{Data: text}})
	if err != nil {
		t.Fatal(err)
	}
	return units
}

// exitOne is the error of a command that exits 1.
func exitOne() error {
	return exec.Command("sh", "-c", "exit 1").Run()
}

func TestFirewallUnitEnablesBlockAllThroughSudo(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[check] = effecttest.Result{Err: exitOne()}
	m.Results[enable] = effecttest.Result{}
	for _, u := range macUnits(t, m) {
		p, err := u.Step.Plan(t.Context())
		if err == nil {
			err = u.Step.Apply(t.Context(), p)
		}
		if err != nil {
			t.Fatalf("%s: %v", u.Name, err)
		}
	}
	if !slices.Contains(m.Calls, "run "+check) || !slices.Contains(m.Calls, "root "+enable) {
		t.Errorf("calls = %q, want the check as the user and the setting through sudo", m.Calls)
	}
}

func TestFirewallUnitChangesNothingWhereBothAreOn(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[check] = effecttest.Result{}
	if _, err := step.Run(t.Context(), kind.Steps(macUnits(t, m)), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(m.Calls, "root "+enable) {
		t.Errorf("calls = %q, want no sudo where the check holds", m.Calls)
	}
}
