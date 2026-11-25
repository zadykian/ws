package firewall_test

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

// The run state's lines: its check, which runs as the user, and its command, which runs as root.
const (
	socketfilterfw = "/usr/libexec/ApplicationFirewall/socketfilterfw"
	inPath         = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
	script         = socketfilterfw + " --getglobalstate | grep -Eq 'State = [12]' && " +
		socketfilterfw + " --getblockall | grep -q 'blocking all'"
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

// TestFirewallCheckReadsTheText runs the check's script on canned output of socketfilterfw's
// get options, as macOS 26 prints it in each state. The options exit 0 in each.
func TestFirewallCheckReadsTheText(t *testing.T) {
	for _, tc := range []struct {
		name, global, blockAll string
		want                   bool
	}{
		{"both on", "Firewall is blocking all non-essential incoming connections. (State = 2)",
			"Firewall is blocking all non-essential incoming connections.", true},
		{"firewall off", "Firewall is disabled. (State = 0)", "Firewall has block all state set to disabled.", false},
		{"firewall off, block all on", "Firewall is disabled. (State = 0)",
			"Firewall is blocking all non-essential incoming connections.", false},
		{"block all off", "Firewall is enabled. (State = 1)", "Firewall has block all state set to disabled.", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			fakeText := "#!/bin/sh\ncase \"$1\" in\n--getglobalstate) echo '" + tc.global + "' ;;\n--getblockall) echo '" +
				tc.blockAll + "' ;;\nesac\n"
			fake := dir + "/socketfilterfw"
			if err := os.WriteFile(fake, []byte(fakeText), 0o700); err != nil { //nolint:gosec // a script that the test runs
				t.Fatal(err)
			}
			//nolint:gosec // the check's own script, pointed at the fake in the test's directory
			err := exec.CommandContext(t.Context(), "sh", "-c", strings.ReplaceAll(script, socketfilterfw, fake)).Run()
			if got := err == nil; got != tc.want {
				t.Errorf("check passes = %v, want %v", got, tc.want)
			}
		})
	}
}
