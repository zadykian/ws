package git_test

import (
	"os"
	"os/exec"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/internal/gitcfg"
	"github.com/zadykian/ws/macos/facts"
	"github.com/zadykian/ws/macos/roles/git"
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

// macUnits loads the role's units on a Mac of the user v, whose ws is /ws, with git's texts. A
// stand-in holds state/ssh-sign-key, which ssh's role has.
func macUnits(t *testing.T, m *effecttest.Machine, texts git.Texts) []kind.Unit {
	fsys := fstest.MapFS{"ssh/units.toml.tmpl": &fstest.MapFile{Data: []byte(
		"[state.ssh-sign-key.run]\nuser = \"v\"\ncreates = [\"/\"]\ncommand = [\"true\"]\n")}}
	for _, name := range []string{"units.toml.tmpl", "files/gitconfig.tmpl", "files/allowed_signers.tmpl",
		"files/ssh-keygen.tmpl"} {
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

// signingKey is the .pub that seward made for git, and the line that trusts it.
const signingKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit"

// keyMade puts the signing key's .pub where seward's unit leaves it, in the user's own ~/.ssh.
func keyMade(m *effecttest.Machine) {
	m.Dirs["/Users/v/.ssh"], m.Owners["/Users/v/.ssh"] = 0o700, [2]int{501, 20}
	m.Files["/Users/v/.ssh/ws-sign.pub"], m.Owners["/Users/v/.ssh/ws-sign.pub"] = []byte(signingKey+" ws-sign\n"), [2]int{501, 20}
}

func TestGitUnitsWriteTheFilesAndTheInclude(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["/ws macos git included"] = effecttest.Result{Err: exitOne()}
	m.Results[ghCheck] = effecttest.Result{}
	keyMade(m)
	texts := git.Texts{Config: "[user]\n\tname = me\n", Keygen: "#!/bin/sh\nexec ssh-keygen \"$@\"\n"}
	units := macUnits(t, m, texts)
	apply(t, units)
	signers := gitcfg.AllowedSigners(gitcfg.Mac(), file.Header("#"), signingKey)
	for path, want := range map[string]string{"/etc/ws/gitconfig": texts.Config, "/etc/ws/allowed_signers": signers,
		"/etc/ws/ssh-keygen": texts.Keygen, "/Users/v/.gitconfig": "[include]\n\tpath = /etc/ws/gitconfig\n"} {
		if got := string(m.Files[path]); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	m.Results["/ws macos git included"] = effecttest.Result{}
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

// TestGitSignersFollowTheSigningKey: the file names the key that seward's unit makes, so it runs
// after that unit, and before the key's .pub exists it holds the header alone.
func TestGitSignersFollowTheSigningKey(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["/ws macos git included"] = effecttest.Result{}
	m.Results[ghCheck] = effecttest.Result{}
	m.Dirs["/Users/v/.ssh"], m.Owners["/Users/v/.ssh"] = 0o700, [2]int{501, 20}
	units := macUnits(t, m, git.Texts{Config: "[gpg]\n"})
	var names []string
	for _, u := range units {
		names = append(names, u.Name)
	}
	if key, signers := slices.Index(names, "state/ssh-sign-key"), slices.Index(names, "file/git-signers"); key < 0 || signers < key {
		t.Errorf("units run in the order %q, want state/ssh-sign-key before file/git-signers", names)
	}
	apply(t, units)
	if got, want := string(m.Files["/etc/ws/allowed_signers"]), file.Header("#")+"\n"; got != want {
		t.Errorf("allowed_signers with no key = %q, want the header alone", got)
	}
}

func TestGitKeygenIsExecutable(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["/ws macos git included"] = effecttest.Result{}
	m.Results[ghCheck] = effecttest.Result{}
	apply(t, macUnits(t, m, git.Texts{Config: "[gpg]\n", Keygen: "#!/bin/sh\nexec ssh-keygen \"$@\"\n"}))
	if m.Modes["/etc/ws/ssh-keygen"]&0o777 != 0o755 {
		t.Errorf("the script's mode = %v, want 0755", m.Modes["/etc/ws/ssh-keygen"])
	}
}

func TestGhUnitCopiesMisesGh(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["/ws macos git included"] = effecttest.Result{}
	m.Results[ghCheck] = effecttest.Result{Err: exitOne()}
	m.Results[ghCopy] = effecttest.Result{}
	apply(t, macUnits(t, m, git.Texts{Config: "[gpg]\n"}))
	if !slices.Contains(m.Calls, "run "+ghCheck) || !slices.Contains(m.Calls, "root "+ghCopy) {
		t.Errorf("calls = %q, want the check as the user and the copy through sudo", m.Calls)
	}
}
