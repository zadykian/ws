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
	_, e := machine(t)
	want := []string{"file/git-config", "file/git-include", "file/git-signers", "guard/git-check",
		"state/git-clones", "state/git-key", "state/git-tokens"}
	got := kind.Names(units(t, e))
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("units = %q, want %q", got, want)
	}
}

func TestGitConfigIsTheStepsFile(t *testing.T) {
	m, e := machine(t)
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
	m, e := machine(t)
	if err := write(t, e, units(t, e, "file/git-config")); err != nil {
		t.Fatal(err)
	}
	settings := gitcfg.Settings(gitcfg.WS(), gitcfg.Inputs{Key: "/root/.ssh/id_ed25519_signing.pub", Gh: "/usr/bin/gh"})
	got := string(m.Files["/etc/ws/gitconfig"])
	if _, rest, ok := strings.Cut(got, "at its top.\n"); !ok || !strings.HasPrefix(rest, settings) {
		t.Errorf("gitconfig = %q, want gitcfg's settings below the header", got)
	}
	if !strings.HasPrefix(got, file.Header("#")) || !strings.HasSuffix(got, "\tpath = /etc/ws/tokens.gitconfig\n") {
		t.Errorf("gitconfig = %q, want the header and the tokens' include", got)
	}
}

func TestGitIncludeGoesOnTopOnce(t *testing.T) {
	m, e := machine(t)
	const rc = "/root/.gitconfig"
	m.Files[rc], m.Owners[rc] = []byte("[core]\n\teditor = vi\n"), [2]int{}
	m.Results[included] = effecttest.Result{Err: exits(t, 1, "")}
	if err := apply(t, units(t, e, "file/git-include")[0]); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[rc]); !strings.HasPrefix(got, include) || !strings.HasSuffix(got, "[core]\n\teditor = vi\n") {
		t.Errorf(".gitconfig = %q, want the include on top of the account's settings", got)
	}
	if m.Owners[rc] != [2]int{} {
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
	m, e := machine(t)
	const rc = "/root/.gitconfig"
	text := "[user]\n\tname = me\n" + include
	m.Files[rc], m.Owners[rc] = []byte(text), [2]int{}
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
	m, e := machine(t)
	m.Results[check] = effecttest.Result{Out: []byte("a clone holds a token\n"), Err: errors.New("exit status 1")}
	m.Results[included] = effecttest.Result{Err: exits(t, 1, "")}
	m.Dirs["/root/.ssh"], m.Owners["/root/.ssh"] = 0o700, [2]int{}
	m.Files["/root/.ssh/id_ed25519_signing"] = []byte("key")
	m.Files["/root/.ssh/id_ed25519_signing.pub"] = []byte(pub)
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
	if _, ok := m.Files["/root/.gitconfig"]; ok {
		t.Error("the include went in after a failed check")
	}
	want := gitcfg.AllowedSigners(gitcfg.WS(), file.Header("#"), "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit")
	if got := string(m.Files["/etc/ws/allowed_signers"]); got != want {
		t.Errorf("allowed_signers after a failed check = %q, want %q", got, want)
	}
}

func TestGitKeyIsMadeOnce(t *testing.T) {
	m, e := machine(t)
	m.Results[keygen] = effecttest.Result{}
	if err := apply(t, units(t, e, "state/git-key")[0]); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(m.Calls, func(call string) bool { return strings.HasSuffix(call, " "+keygen) }) {
		t.Errorf("calls %q, want ssh-keygen as root", m.Calls)
	}
	m.Files["/root/.ssh/id_ed25519_signing"] = []byte("key")
	if got := changes(t, units(t, e, "state/git-key")[0]); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestGitSignersTrustTheKey(t *testing.T) {
	m, e := machine(t)
	const p = "/etc/ws/allowed_signers"
	if err := apply(t, units(t, e, "file/git-signers")[0]); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[p]); got != file.Header("#")+"\n" {
		t.Errorf("without the key: %s = %q, want the header alone", p, got)
	}
	m.Dirs["/root/.ssh"], m.Owners["/root/.ssh"] = 0o700, [2]int{}
	m.Files["/root/.ssh/id_ed25519_signing.pub"] = []byte(pub)
	if err := apply(t, units(t, e, "file/git-signers")[0]); err != nil {
		t.Fatal(err)
	}
	want := gitcfg.AllowedSigners(gitcfg.WS(), file.Header("#"), "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit")
	if got := string(m.Files[p]); got != want || m.Modes[p] != 0o644 {
		t.Errorf("%s = %q, mode %o; want %q, 644", p, got, m.Modes[p], want)
	}
	if got := changes(t, units(t, e, "file/git-signers")[0]); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestGitTokensAndClonesRunAsRoot(t *testing.T) {
	m, e := machine(t)
	for _, name := range []string{"tokens", "clones"} {
		line := "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin " + self +
			" server git " + name
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
