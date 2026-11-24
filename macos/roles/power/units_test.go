package power_test

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

// The run state's lines: the check runs as the user, the command as root through sudo.
const (
	inPath        = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
	displayScript = "pmset -g custom | awk '/^AC Power:/ {ac = 1; next} /^[^ ]/ {ac = 0} " +
		"ac && $1 == \"displaysleep\" && $2 == 0 {found = 1} END {exit !found}'"
	displayCheck = inPath + "sh -c " + displayScript
	displayRun   = inPath + "pmset -c displaysleep 0"
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
		Data: facts.Mac{User: "v", Home: "/Users/v"}}, fstest.MapFS{"power/units.toml.tmpl": &fstest.MapFile{Data: text}})
	if err != nil {
		t.Fatal(err)
	}
	return units
}

// exitOne is the error of a command that exits 1.
func exitOne() error {
	return exec.Command("sh", "-c", "exit 1").Run()
}

func TestDisplayUnitChangesThroughSudoWhereTheCheckFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[displayCheck] = effecttest.Result{Err: exitOne()}
	m.Results[displayRun] = effecttest.Result{}
	// The check keeps failing, as the fake machine changes nothing, so the unit plans and applies once.
	for _, u := range macUnits(t, m) {
		p, err := u.Step.Plan(t.Context())
		if err == nil {
			err = u.Step.Apply(t.Context(), p)
		}
		if err != nil {
			t.Fatalf("%s: %v", u.Name, err)
		}
	}
	for _, want := range []string{"run " + displayCheck, "root " + displayRun} {
		if !slices.Contains(m.Calls, want) {
			t.Errorf("calls = %q, want %q", m.Calls, want)
		}
	}
}

func TestDisplayUnitAsksNoSudoWhereTheCheckHolds(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[displayCheck] = effecttest.Result{}
	if _, err := step.Run(t.Context(), kind.Steps(macUnits(t, m)), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root ") {
			t.Errorf("calls = %q, want no sudo where the check holds", m.Calls)
		}
	}
}

// runCheck runs script, with the fake programs of dir first on PATH, and reports whether it passes.
func runCheck(t *testing.T, dir, script string) bool {
	cmd := exec.CommandContext(t.Context(), "sh", "-c", script) //nolint:gosec // the unit's own script, run on fakes
	cmd.Env = []string{"PATH=" + dir + ":/usr/bin:/bin"}
	return cmd.Run() == nil
}

// fake writes the program name into dir, a script that prints out and exits with status.
func fake(t *testing.T, dir, name, out string, status int) {
	text := "#!/bin/sh\ncat <<'END'\n" + out + "END\nexit " + strconv.Itoa(status) + "\n"
	if err := os.WriteFile(dir+"/"+name, []byte(text), 0o700); err != nil { //nolint:gosec // a script that the test runs
		t.Fatal(err)
	}
}

// TestDisplayCheckReadsTheACSection runs the check on canned output of pmset -g custom, as macOS 26
// prints it: a laptop lists the battery first, and a desktop has the AC section alone.
func TestDisplayCheckReadsTheACSection(t *testing.T) {
	const battery = "Battery Power:\n displaysleep         %s\n sleep                1\n"
	const ac = "AC Power:\n Sleep On Power Button 1\n displaysleep         %s\n sleep                0\n"
	for _, tc := range []struct {
		name, out string
		want      bool
	}{
		{"on power, never", fmt.Sprintf(ac, "0"), true},
		{"on power, after ten minutes", fmt.Sprintf(ac, "10"), false},
		{"battery's zero alone", fmt.Sprintf(battery, "0") + fmt.Sprintf(ac, "10"), false},
		{"both zero", fmt.Sprintf(battery, "0") + fmt.Sprintf(ac, "0"), true},
		{"battery after ac", fmt.Sprintf(ac, "10") + fmt.Sprintf(battery, "0"), false},
		{"no setting", "AC Power:\n sleep 0\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			fake(t, dir, "pmset", tc.out, 0)
			if got := runCheck(t, dir, displayScript); got != tc.want {
				t.Errorf("check passes = %v, want %v", got, tc.want)
			}
		})
	}
}
