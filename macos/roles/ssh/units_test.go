package ssh_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/macos/facts"
)

// userPath is the PATH that a run state's command of the user's runs with.
const userPath = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "

// unitsConfig is the text that the facts give /etc/ws/ssh_config.
const unitsConfig = "# Written by github.com/zadykian/ws\nHost ws\n"

// macUnits loads the role's units on a Mac of the user v, whose ws is /ws.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/ssh_config.tmpl"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["ssh/"+name] = &fstest.MapFile{Data: text}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"], m.Dirs["/etc"] = 0o750, 0o755
	m.Owners["/Users/v"] = [2]int{501, 20}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, Self: "/ws", System: kind.MacOS, User: "v",
		Data: facts.Mac{User: "v", Home: "/Users/v", SSH: facts.SSH{Config: unitsConfig}}}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

func TestSSHUnitsWriteTheHostsAndTheInclude(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[userPath+"/ws macos keys --check"] = effecttest.Result{}
	m.Results[userPath+"mkdir -m 700 /Users/v/.ssh"] = effecttest.Result{}
	m.Results["ssh -G -F /etc/ws/ssh_config.ws-check localhost"] = effecttest.Result{}
	m.Results["/ws macos ssh included"] = effecttest.Result{Err: exited("exit 1")}
	m.Files["/Users/v/.ssh/config"], m.Owners["/Users/v/.ssh/config"] = []byte("Host *\n    User me\n"), [2]int{501, 20}
	m.Dirs["/Users/v/.ssh"], m.Owners["/Users/v/.ssh"] = 0o700, [2]int{501, 20}
	units := macUnits(t, m)
	// The fake's has answers alike whatever the file holds, so each unit applies once, and has
	// answers 0 after.
	for _, u := range units {
		p, err := u.Step.Plan(t.Context())
		if err == nil {
			err = u.Step.Apply(t.Context(), p)
		}
		if err != nil {
			t.Fatalf("%s: %v", u.Name, err)
		}
	}
	if got := string(m.Files["/etc/ws/ssh_config"]); got != unitsConfig {
		t.Errorf("/etc/ws/ssh_config = %q, want the facts' text", got)
	}
	want := "Include /etc/ws/ssh_config\n\nHost *\n    User me\n"
	if got := string(m.Files["/Users/v/.ssh/config"]); got != want {
		t.Errorf("~/.ssh/config = %q, want %q", got, want)
	}
	m.Results["/ws macos ssh included"] = effecttest.Result{}
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

func TestSSHIncludeRefusesAnIncludeBelow(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/Users/v/.ssh/config"] = []byte("Host *\n    User me\nInclude /etc/ws/ssh_config\n")
	m.Results["/ws macos ssh included"] = effecttest.Result{Err: exited("echo line 3 below >&2; exit 2")}
	for _, u := range macUnits(t, m) {
		if u.Name != "file/ssh-include" {
			continue
		}
		if _, err := u.Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "line 3 below") {
			t.Errorf("plan = %v, want has's refusal", err)
		}
	}
}

// A dotfiles manager's link at ~/.ssh/config whose file holds the Include is in sync: the user
// reads through it. The fake's file at the path stands for what that read gives.
func TestSSHIncludeTakesALinkThatHoldsIt(t *testing.T) {
	m := effecttest.NewMachine()
	m.Links["/Users/v/.ssh/config"] = "/Users/v/dots/ssh_config"
	m.Files["/Users/v/.ssh/config"] = []byte("Include /etc/ws/ssh_config\n")
	m.Results["/ws macos ssh included"] = effecttest.Result{}
	for _, u := range macUnits(t, m) {
		if u.Name != "file/ssh-include" {
			continue
		}
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("plan = %v, %v; want none", p.Changes, err)
		}
	}
}

// exited is the error of a shell that runs script, with its exit status and standard error.
func exited(script string) error {
	_, err := exec.Command("sh", "-c", script).Output() //nolint:gosec // the test's own script
	return err
}
