package git

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

func gitFixture() (*effecttest.Machine, stdinMachine, *effecttest.Out, Config) {
	m := effecttest.NewMachine()
	m.Results["git config --file - --name-only --list"] = effecttest.Result{}
	m.Results["git config --file - --type path --get-all include.path"] = effecttest.Result{}
	m.Results["git config --file - --get-regexp "+helperPattern] = effecttest.Result{}
	sm := stdinMachine{Machine: m, answers: map[string]string{}, paths: map[string]string{}, helpers: map[string]string{}}
	out := &effecttest.Out{}
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	return m, sm, out, Config{M: sm, Out: out, A: a,
		Key: "/home/ws/.ssh/id_ed25519_signing.pub"}
}

func apply(t *testing.T, g Config) []resource.Change {
	got, err := g.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return got
}

func TestGitconfigIsTheRolesRendering(t *testing.T) {
	_, _, _, g := gitFixture()
	want, err := os.ReadFile("testdata/gitconfig")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := g.content(); err != nil || string(got) != string(want) {
		t.Errorf("testdata/gitconfig: differs from the role's rendering (%v)", err)
	}
}

func TestGitconfigIsRootsAndTheUsersFileHoldsTheInclude(t *testing.T) {
	m, _, _, g := gitFixture()
	apply(t, g)
	want, _ := g.content()
	if string(m.Files["/etc/ws/gitconfig"]) != string(want) || m.Modes["/etc/ws/gitconfig"] != 0o644 || m.Owners["/etc/ws/gitconfig"] != [2]int{0, 0} {
		t.Errorf("/etc/ws/gitconfig %o %v", m.Modes["/etc/ws/gitconfig"], m.Owners["/etc/ws/gitconfig"])
	}
	const rc = "/home/ws/.gitconfig"
	if string(m.Files[rc]) != "[include]\n\tpath = /etc/ws/gitconfig\n" || m.Modes[rc] != 0o644 || m.Owners[rc] != [2]int{1000, 1000} {
		t.Errorf("%s %q %o %v", rc, m.Files[rc], m.Modes[rc], m.Owners[rc])
	}
}

func TestGitconfigKeepsTheUsersFileAndNamesItsOwnSettings(t *testing.T) {
	m, sm, out, g := gitFixture()
	const rc = "/home/ws/.gitconfig"
	old := "[user]\n\temail = a@b.c\n[http]\n\textraheader = hunter2\n"
	m.Files[rc] = []byte(old)
	sm.answers["[user]"] = "user.email\nhttp.extraheader\n"
	got := apply(t, g)
	if !strings.HasPrefix(string(m.Files[rc]), "[include]\n\tpath = /etc/ws/gitconfig\n\n[user]") {
		t.Errorf("%s = %q", rc, m.Files[rc])
	}
	var kept string
	for path, text := range m.Files {
		if strings.HasPrefix(path, rc+".") && string(text) == old {
			kept = path
		}
	}
	if kept == "" {
		t.Errorf("no kept copy among %v", m.Files)
	}
	for _, c := range got {
		if strings.Contains(c.String(), "hunter2") {
			t.Errorf("a change shows %q", c.String())
		}
	}
	if all := strings.Join(out.Lines, "\n"); !strings.Contains(all, "own settings of user.email, below the include") {
		t.Errorf("said %q", all)
	}
}

func TestGitconfigSecondPlanIsEmpty(t *testing.T) {
	_, sm, _, g := gitFixture()
	apply(t, g)
	sm.paths["[include]"] = "/etc/ws/gitconfig\n"
	sm.answers["[include]"] = "include.path\n"
	if got, err := g.Plan(context.Background()); err != nil || len(got) != 0 {
		t.Fatalf("plan %v, %v", got, err)
	}
}

func TestGitconfigRefusesAnIncludeBelowTheUsersSettings(t *testing.T) {
	m, sm, _, g := gitFixture()
	m.Files["/home/ws/.gitconfig"] = []byte("[user]\n\tname = x\n[include]\n\tpath = /etc/ws/gitconfig\n")
	sm.answers["[user]"] = "user.name\ninclude.path\n"
	sm.paths["[user]"] = "/etc/ws/gitconfig\n"
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "below its own settings") {
		t.Errorf("err = %v", err)
	}
}
