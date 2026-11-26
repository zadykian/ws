package desktop_test

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

// The run state's lines: the check and the command run as the user, with Homebrew's bin first.
const (
	stone  = "/System/Library/Desktop Pictures/Solid Colors/Stone.png"
	inPath = "env PATH=/opt/homebrew/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
	script = "wallpaper get | awk '$0 != \"" + stone + "\" {bad = 1} END {exit bad || NR == 0}'"
	check  = inPath + "sh -c " + script
	set    = inPath + "wallpaper set " + stone
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
		Data: facts.Mac{User: "v", Home: "/Users/v"}}, fstest.MapFS{"desktop/units.toml.tmpl": &fstest.MapFile{Data: text}})
	if err != nil {
		t.Fatal(err)
	}
	return units
}

func TestWallpaperIsSetAsTheUserWhereTheCheckFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[check] = effecttest.Result{Err: exec.Command("sh", "-c", "exit 1").Run()}
	m.Results[set] = effecttest.Result{}
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
	if got, want := strings.Join(m.Calls, "\n"), "run "+check+"\nrun "+set; !strings.Contains(got, want) {
		t.Errorf("calls = %q, want the check then %q, both as the user", m.Calls, "run "+set)
	}
}

func TestWallpaperAsksNoSudoWhereTheCheckHolds(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[check] = effecttest.Result{}
	if _, err := step.Run(t.Context(), kind.Steps(macUnits(t, m)), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root ") || c == "run "+set {
			t.Errorf("calls = %q, want the check alone, and no sudo", m.Calls)
		}
	}
}

// TestCheckReadsEachScreen runs the check on canned output of wallpaper get, a line per screen,
// and on a failure. The pipe has no pipefail, so awk's exit alone decides.
func TestCheckReadsEachScreen(t *testing.T) {
	other := "/System/Library/Desktop Pictures/Solid Colors/Black.png"
	for _, tc := range []struct {
		name, out string
		status    int
		want      bool
	}{
		{"one screen on Stone", stone + "\n", 0, true},
		{"two screens on Stone", stone + "\n" + stone + "\n", 0, true},
		{"one of two on another", stone + "\n" + other + "\n", 0, false},
		{"another picture", other + "\n", 0, false},
		{"no screen", "", 0, false},
		{"wallpaper exits 1 after Stone's path", stone + "\n", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			text := "#!/bin/sh\ncat <<'END'\n" + tc.out + "END\nexit " + strconv.Itoa(tc.status) + "\n"
			if err := os.WriteFile(dir+"/wallpaper", []byte(text), 0o700); err != nil { //nolint:gosec // a script that the test runs
				t.Fatal(err)
			}
			cmd := exec.CommandContext(t.Context(), "sh", "-c", script)
			cmd.Env = []string{"PATH=" + dir + ":/usr/bin:/bin"}
			if got := cmd.Run() == nil; got != tc.want {
				t.Errorf("check passes = %v, want %v", got, tc.want)
			}
		})
	}
}
