package git_test

import (
	"os"
	"os/exec"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/roles/client/facts"
	"github.com/zadykian/ws/roles/client/macos/git"
)

// misesGh is where mise installs the tree's gh.
const misesGh = "/Users/v/.local/share/mise/installs/gh/2.104.0/gh_2.104.0_macOS_arm64/bin/gh"

// The run state's lines of gh: its check, which runs as the user, and its command, which runs as root.
const (
	ghCheck = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin sh -c cmp -s \"$1\" " +
		"/usr/local/bin/gh && [ \"$(stat -f %u:%g:%Lp /usr/local/bin/gh)\" = 0:0:755 ] sh " + misesGh
	ghCopy = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin sh -c test -x \"$1\" && " +
		"mkdir -p /usr/local/bin && install -o root -g wheel -m 0755 \"$1\" /usr/local/bin/.gh.new && " +
		"mv -f /usr/local/bin/.gh.new /usr/local/bin/gh sh " + misesGh
)

// macUnits loads the role's units, but state/keys, which ssh's role holds, on a Mac of the user v,
// whose ws is /ws, with git's texts.
func macUnits(t *testing.T, m *effecttest.Machine, texts git.Texts) []kind.Unit {
	fsys := fstest.MapFS{"keys/units.toml.tmpl": &fstest.MapFile{Data: []byte(
		"[state.keys.run]\nuser = \"v\"\ncreates = [\"/\"]\ncommand = [\"true\"]\n")}}
	for _, name := range []string{"units.toml.tmpl", "files/gitconfig.tmpl", "files/allowed_signers.tmpl",
		"files/secretive-ssh-keygen.tmpl"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["git/"+name] = &fstest.MapFile{Data: text}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"], m.Dirs["/etc"], m.Dirs["/"] = 0o750, 0o755, 0o755
	m.Owners["/Users/v"] = [2]int{501, 20}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, Self: "/ws", System: kind.MacOS, User: "v",
		Data: facts.Mac{User: "v", Home: "/Users/v", Git: facts.Git{Texts: texts, GhFrom: misesGh}}}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

// apply plans and applies each unit once, as the fake's has answers alike whatever the file holds.
func apply(t *testing.T, units []kind.Unit) {
	for _, u := range units {
		p, err := u.Step.Plan(t.Context())
		if err == nil {
			err = u.Step.Apply(t.Context(), p)
		}
		if err != nil {
			t.Fatalf("%s: %v", u.Name, err)
		}
	}
}

// exitOne is the error of a command that exits 1.
func exitOne() error {
	return exec.Command("sh", "-c", "exit 1").Run()
}

func TestGitUnitsWriteTheFilesAndTheInclude(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["/ws client git included"] = effecttest.Result{Err: exitOne()}
	m.Results[ghCheck] = effecttest.Result{}
	m.Files["/etc/ws/secretive-ssh-keygen"] = []byte("# Written by github.com/zadykian/ws\n")
	texts := git.Texts{Config: "[user]\n\tname = me\n", Signers: "me@zadykian.dev ssh-ed25519 AAAA\n"}
	units := macUnits(t, m, texts)
	apply(t, units)
	for path, want := range map[string]string{"/etc/ws/gitconfig": texts.Config, "/etc/ws/allowed_signers": texts.Signers,
		"/Users/v/.gitconfig": "[include]\n\tpath = /etc/ws/gitconfig\n"} {
		if got := string(m.Files[path]); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if _, ok := m.Files["/etc/ws/secretive-ssh-keygen"]; ok {
		t.Error("the script stays for a key in a file, want it gone")
	}
	m.Results["/ws client git included"] = effecttest.Result{}
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

func TestGitKeygenIsExecutable(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["/ws client git included"] = effecttest.Result{}
	m.Results[ghCheck] = effecttest.Result{}
	apply(t, macUnits(t, m, git.Texts{Config: "[gpg]\n", Signers: "x\n", Keygen: "#!/bin/sh\nexec ssh-keygen \"$@\"\n"}))
	if m.Modes["/etc/ws/secretive-ssh-keygen"]&0o777 != 0o755 {
		t.Errorf("the script's mode = %v, want 0755", m.Modes["/etc/ws/secretive-ssh-keygen"])
	}
}

func TestGhUnitCopiesMisesGh(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["/ws client git included"] = effecttest.Result{}
	m.Results[ghCheck] = effecttest.Result{Err: exitOne()}
	m.Results[ghCopy] = effecttest.Result{}
	apply(t, macUnits(t, m, git.Texts{Config: "[gpg]\n", Signers: "x\n"}))
	if !slices.Contains(m.Calls, "run "+ghCheck) || !slices.Contains(m.Calls, "root "+ghCopy) {
		t.Errorf("calls = %q, want the check as the user and the copy through sudo", m.Calls)
	}
}
