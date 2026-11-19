package git_test

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/internal/gitcfg"
	"github.com/zadykian/ws/roles/server/values"
)

const (
	self     = "/usr/local/bin/ws"
	check    = self + " server ubuntu git check"
	included = self + " server ubuntu git included"
	include  = "[include]\n\tpath = /etc/ws/gitconfig\n"
	pub      = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit ws@box\n"
)

// role is git's spec and its templates, as kind.Load finds a role's.
func role(t *testing.T) fstest.MapFS {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/ws.gitconfig.tmpl", "files/allowed_signers.tmpl"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["git/"+name] = &fstest.MapFile{Data: text}
	}
	return fsys
}

// machine is a machine with the account ws, 1000:1000, and /etc/ws, and its env for the values of
// text, whose check passes.
func machine(t *testing.T, text string) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["getent passwd ws"] = effecttest.Result{Out: []byte("ws:x:1000:1000::/home/ws:/bin/bash\n")}
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/home/ws"], m.Owners["/home/ws"] = 0o750, [2]int{1000, 1000}
	m.Dirs["/root"], m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o700, 0o755, 0o755
	m.Results[check] = effecttest.Result{}
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Self: self}
}

// units loads git's units on e, and keeps those named.
func units(t *testing.T, e kind.Env, names ...string) []kind.Unit {
	all, err := kind.Load(e, role(t))
	if err != nil {
		t.Fatal(err)
	}
	return slices.DeleteFunc(all, func(u kind.Unit) bool { return len(names) > 0 && !slices.Contains(names, u.Name) })
}

// write runs the units as ws unit write does.
func write(t *testing.T, e kind.Env, us []kind.Unit) error {
	_, err := step.Run(t.Context(), kind.Steps(us), e.Out, false)
	return err
}

// apply plans the unit and applies its changes, as ws unit write does, without the check that a
// second plan holds none, which the fake's fixed answers would fail.
func apply(t *testing.T, u kind.Unit) error {
	p, err := u.Step.Plan(t.Context())
	if err != nil {
		return err
	}
	return u.Step.Apply(t.Context(), p)
}

// changes returns the headlines of the unit's plan.
func changes(t *testing.T, u kind.Unit) []string {
	p, err := u.Step.Plan(t.Context())
	if err != nil {
		t.Fatalf("plan %s: %v", u.Name, err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, step.Headline(c))
	}
	return got
}

// exits is the error of a command that exits with code, saying why on its standard error.
func exits(t *testing.T, code int, why string) error {
	_, err := exec.Command("sh", "-c", `echo "$1" >&2; exit "$2"`, "sh", why, strconv.Itoa(code)).Output() //nolint:gosec // G204: the test's own sh, its words as arguments
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("sh: %v, want an exit status", err)
	}
	return err
}

// keygen is the command line that makes the key of an account's default path, as the fake
// records it.
func keygen(user, home string) string {
	key := home + "/.ssh/id_ed25519_signing"
	sh := "sh -c mkdir -p -m 700 \"${1%/*}\" && ssh-keygen -q -t ed25519 -N '' -f \"$1\" sh " + key
	env := "env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin " + sh
	if user == "root" {
		return "env HOME=/root " + env
	}
	return "runuser -u " + user + " -- env -i -C " + home + " HOME=" + home + " " + account.SystemPath + " " + env
}

func TestGitUnits(t *testing.T) {
	_, e := machine(t, "WS_USER: ws\n")
	want := []string{"file/git-config", "file/git-include", "file/git-signers", "guard/git-check",
		"state/git-clones", "state/git-key", "state/git-tokens"}
	got := kind.Names(units(t, e))
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("units = %q, want %q", got, want)
	}
}

func TestGitConfigIsTheStepsFile(t *testing.T) {
	m, e := machine(t, "WS_USER: ws\n")
	if err := write(t, e, units(t, e, "file/git-config")); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/gitconfig")
	if err != nil {
		t.Fatal(err)
	}
	const p = "/etc/ws/gitconfig"
	if got := m.Files[p]; string(got) != string(want) || m.Modes[p] != 0o644 || m.Owners[p] != [2]int{} {
		t.Errorf("%s = %q, mode %o, owner %v; want the step's file, root's, 644", p, got, m.Modes[p], m.Owners[p])
	}
}

func TestGitConfigOfRootIsGitcfgsSettings(t *testing.T) {
	m, e := machine(t, "WS_USER: root\n")
	if err := write(t, e, units(t, e, "file/git-config")); err != nil {
		t.Fatal(err)
	}
	settings := gitcfg.Settings(gitcfg.Inputs{Key: "/root/.ssh/id_ed25519_signing.pub", Gh: "/usr/bin/gh"})
	got := string(m.Files["/etc/ws/gitconfig"])
	if _, rest, ok := strings.Cut(got, "at its top.\n"); !ok || !strings.HasPrefix(rest, settings) {
		t.Errorf("gitconfig = %q, want gitcfg's settings below the header", got)
	}
	if !strings.HasPrefix(got, file.Header("#")) || !strings.HasSuffix(got, "\tpath = /etc/ws/tokens.gitconfig\n") {
		t.Errorf("gitconfig = %q, want the header and the tokens' include", got)
	}
}

func TestGitIncludeGoesOnTopOnce(t *testing.T) {
	m, e := machine(t, "WS_USER: ws\n")
	const rc = "/home/ws/.gitconfig"
	m.Files[rc], m.Owners[rc] = []byte("[core]\n\teditor = vi\n"), [2]int{1000, 1000}
	m.Results[included] = effecttest.Result{Err: exits(t, 1, "")}
	if err := apply(t, units(t, e, "file/git-include")[0]); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[rc]); !strings.HasPrefix(got, include) || !strings.HasSuffix(got, "[core]\n\teditor = vi\n") {
		t.Errorf(".gitconfig = %q, want the include on top of the account's settings", got)
	}
	if m.Owners[rc] != [2]int{1000, 1000} {
		t.Errorf("owner %v, want the account's", m.Owners[rc])
	}
	m.Results[included] = effecttest.Result{}
	if got := changes(t, units(t, e, "file/git-include")[0]); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	if i := slices.Index(m.Inputs, string(m.Files[rc])); i < 0 {
		t.Errorf("inputs %q, want the file's text on included's stdin", m.Inputs)
	}
}

func TestGitIncludeRefusedLeavesTheFile(t *testing.T) {
	m, e := machine(t, "WS_USER: ws\n")
	const rc = "/home/ws/.gitconfig"
	text := "[user]\n\tname = me\n" + include
	m.Files[rc], m.Owners[rc] = []byte(text), [2]int{1000, 1000}
	m.Results[included] = effecttest.Result{Err: exits(t, 2, "the include sits below")}
	err := apply(t, units(t, e, "file/git-include")[0])
	if err == nil || !strings.Contains(err.Error(), "the include sits below") {
		t.Errorf("write = %v, want included's refusal", err)
	}
	if string(m.Files[rc]) != text {
		t.Errorf(".gitconfig = %q, want it as it was", m.Files[rc])
	}
}

// A failed check stops the include, the tokens and the clones, but not the signing key's units,
// which run before it.
func TestGitCheckStopsAllButSigning(t *testing.T) {
	m, e := machine(t, "WS_USER: ws\n")
	m.Results[check] = effecttest.Result{Out: []byte("a clone holds a token\n"), Err: errors.New("exit status 1")}
	m.Results[included] = effecttest.Result{Err: exits(t, 1, "")}
	m.Dirs["/home/ws/.ssh"], m.Owners["/home/ws/.ssh"] = 0o700, [2]int{1000, 1000}
	m.Files["/home/ws/.ssh/id_ed25519_signing"] = []byte("key")
	m.Files["/home/ws/.ssh/id_ed25519_signing.pub"] = []byte(pub)
	us := units(t, e)
	names := kind.Names(us)
	if g := slices.Index(names, "guard/git-check"); slices.Index(names, "state/git-key") > g || slices.Index(names, "file/git-signers") > g {
		t.Errorf("order = %q, want the key and its signers before the check", names)
	}
	if err := write(t, e, us); err == nil {
		t.Fatal("write = nil, want the check's failure")
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "git included") || strings.Contains(c, "git tokens") || strings.Contains(c, "git clones") {
			t.Errorf("call %q after a failed check", c)
		}
	}
	if _, ok := m.Files["/home/ws/.gitconfig"]; ok {
		t.Error("the include went in after a failed check")
	}
	want := gitcfg.AllowedSigners(file.Header("#"), "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit")
	if got := string(m.Files["/etc/ws/allowed_signers"]); got != want {
		t.Errorf("allowed_signers after a failed check = %q, want %q", got, want)
	}
}

func TestGitKeyIsMadeOnceWithoutSystemd(t *testing.T) {
	for _, c := range []struct{ values, user, home string }{
		{"WS_USER: ws\n", "ws", "/home/ws"}, {"WS_USER: root\n", "root", "/root"},
	} {
		m, e := machine(t, c.values)
		line := keygen(c.user, c.home)
		m.Results[line] = effecttest.Result{}
		if err := apply(t, units(t, e, "state/git-key")[0]); err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(m.Calls, func(call string) bool { return strings.HasSuffix(call, " "+line) }) {
			t.Errorf("%s: calls %q, want ssh-keygen as the account, though systemd is not running", c.user, m.Calls)
		}
		m.Files[c.home+"/.ssh/id_ed25519_signing"] = []byte("key")
		if got := changes(t, units(t, e, "state/git-key")[0]); got != nil {
			t.Errorf("%s: second plan = %q, want none", c.user, got)
		}
	}
}

func TestGitSignersTrustTheKey(t *testing.T) {
	m, e := machine(t, "WS_USER: ws\n")
	const p = "/etc/ws/allowed_signers"
	if err := apply(t, units(t, e, "file/git-signers")[0]); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[p]); got != file.Header("#")+"\n" {
		t.Errorf("without the key: %s = %q, want the header alone", p, got)
	}
	m.Dirs["/home/ws/.ssh"], m.Owners["/home/ws/.ssh"] = 0o700, [2]int{1000, 1000}
	m.Files["/home/ws/.ssh/id_ed25519_signing.pub"] = []byte(pub)
	if err := apply(t, units(t, e, "file/git-signers")[0]); err != nil {
		t.Fatal(err)
	}
	want := gitcfg.AllowedSigners(file.Header("#"), "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit")
	if got := string(m.Files[p]); got != want || m.Modes[p] != 0o644 {
		t.Errorf("%s = %q, mode %o; want %q, 644", p, got, m.Modes[p], want)
	}
	if got := changes(t, units(t, e, "file/git-signers")[0]); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestGitTokensAndClonesRunAsRootWithoutSystemd(t *testing.T) {
	m, e := machine(t, "WS_USER: ws\n")
	for _, name := range []string{"tokens", "clones"} {
		line := "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin " + self +
			" server ubuntu git " + name
		m.Results[line+" --check"] = effecttest.Result{Err: errors.New("exit status 1")}
		m.Results[line] = effecttest.Result{}
	}
	us := units(t, e, "guard/git-check", "state/git-tokens", "state/git-clones")
	if got, want := kind.Names(us), []string{"guard/git-check", "state/git-tokens", "state/git-clones"}; !slices.Equal(got, want) {
		t.Errorf("order = %q, want %q", got, want)
	}
	for _, u := range us[1:] {
		if err := apply(t, u); err != nil {
			t.Fatal(err)
		}
	}
	var ran []string
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "run env HOME=/root") && !strings.HasSuffix(c, "--check") {
			ran = append(ran, c[strings.LastIndex(c, " ")+1:])
		}
	}
	if !slices.Equal(ran, []string{"tokens", "clones"}) {
		t.Errorf("ran %q, want the tokens and the clones as root; calls %q", ran, m.Calls)
	}
}
