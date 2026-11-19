package git_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/internal/gitcfg"
)

func TestGitUnits(t *testing.T) {
	_, e := machine(t, "WS_USER: ws\n")
	want := []string{"file/git-config", "file/git-include", "file/git-signers", "guard/git-check",
		"state/git-clones", "state/git-key", "state/git-tokens", "state/git-tokens-group"}
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
	// The run takes the frame's unit too, which git's chain waits on and which skips without systemd.
	us, err := kind.Closure(loaded(t, e), kind.Names(units(t, e)))
	if err != nil {
		t.Fatal(err)
	}
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
