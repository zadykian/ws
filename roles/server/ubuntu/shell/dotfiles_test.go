package shell

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
)

var testAccount = account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}

func TestDotfilesAreWsFilesThenTheUsersLines(t *testing.T) {
	m := effecttest.NewMachine()
	got, err := dotfiles(m, testAccount, "amd64")
	if err != nil || len(got) != 5 {
		t.Fatalf("got %d parts, %v", len(got), err)
	}
	if d, ok := got[0].(file.PublicDir); !ok || d.Path != "/etc/ws" {
		t.Errorf("part 0 = %+v, want /etc/ws", got[0])
	}
	for i, name := range []string{"bashrc", "profile"} {
		want, err := os.ReadFile("testdata/" + name) //nolint:gosec // the name is one of the test's own constants
		if err != nil {
			t.Fatal(err)
		}
		f, ok := got[i+1].(file.Public)
		if !ok || f.Path != "/etc/ws/"+name || f.Mode != 0o644 || !bytes.Equal(f.Content, want) {
			t.Errorf("part %d = %+v, want /etc/ws/%s as the role's rendering", i+1, f.Path, name)
		}
	}
	for i, name := range []string{".bashrc", ".profile"} {
		l, ok := got[i+3].(file.HomeLines)
		if !ok || l.Rules.Path != "/home/ws/"+name || !l.Rules.Top {
			t.Errorf("part %d = %+v, want the line at the top of ~/%s", i+3, got[i+3], name)
		}
	}
}

func TestTheUsersFilesGetOneLineAtTheTopAndKeepTheirCopy(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/home/ws/.bashrc"] = []byte("# stock\n[ -z \"$PS1\" ] && return\n")
	got, err := dotfiles(m, testAccount, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	rc := got[3].(file.HomeLines)
	for range 2 {
		changes, err := rc.Plan(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range changes {
			if err := c.Apply(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got, want := string(m.Files["/home/ws/.bashrc"]), ". /etc/ws/bashrc\n\n# stock\n[ -z \"$PS1\" ] && return\n"; got != want {
		t.Errorf(".bashrc = %q, want %q", got, want)
	}
	kept := 0
	for p := range m.Files {
		if strings.HasPrefix(p, "/home/ws/.bashrc.") && strings.HasSuffix(p, "~") {
			kept++
		}
	}
	if kept != 1 {
		t.Errorf("%d copies of the old file, want 1", kept)
	}
	if again, err := rc.Plan(t.Context()); err != nil || len(again) != 0 {
		t.Errorf("a second plan = %v, %v, want none", again, err)
	}
}

func TestAMissingProfileIsMadeWithTheLineAlone(t *testing.T) {
	m := effecttest.NewMachine()
	got, err := dotfiles(m, testAccount, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	changes, err := got[4].(file.HomeLines).Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range changes {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := string(m.Files["/home/ws/.profile"]), ". /etc/ws/profile\n"; got != want {
		t.Errorf(".profile = %q, want %q", got, want)
	}
}

func TestDotnetArch(t *testing.T) {
	for arch, want := range map[string]string{"arm64": "arm64", "amd64": "x64", "riscv64": "x64"} {
		if got := dotnetArch(arch); got != want {
			t.Errorf("%s: %s, want %s", arch, got, want)
		}
	}
}
