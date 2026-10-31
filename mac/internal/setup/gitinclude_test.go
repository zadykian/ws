package setup_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestGitPutsTheIncludeAtTheTopOfGitconfig(t *testing.T) {
	m, out, g := gitWithConfig("[user]\n    name = X\n", "", "user.name\n")
	m.Results["git config --file "+gitConfigFile+" --type path --get-all include.path"] =
		effecttest.Result{Err: exitOne(t)}
	m.Results["cp -p "+gitConfigFile+" "+gitConfigFile+".20260102030405~"] = effecttest.Result{}
	p := mustPlan(t, g)
	if err := g.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got, want := string(m.Files[gitConfigFile]), includeText+"\n[user]\n    name = X\n"; got != want {
		t.Errorf("~/.gitconfig = %q, want %q", got, want)
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "own settings of user.name, below the include") {
		t.Errorf("no warning of user.name: %q", out.Lines)
	}
}

func TestGitMakesGitconfigWhereMissing(t *testing.T) {
	_, _, g := gitWithConfig("", "", "")
	p := mustPlan(t, g)
	if len(p.Changes) != 3 || !strings.Contains(p.Changes[2].String(), "+    path = ~/.config/ws/gitconfig") {
		t.Errorf("Plan = %v; want the two files and a new ~/.gitconfig", p.Changes)
	}
}

func exitOne(t *testing.T) error {
	err := exec.Command("sh", "-c", "exit 1").Run()
	if err == nil {
		t.Fatal("sh -c 'exit 1' succeeded")
	}
	return err
}

func TestGitWarnsOfXDGSettingsThatLose(t *testing.T) {
	m, out, g := gitWithConfig("x", "", "user.name\n")
	xdg := gitHome + "/.config/git/config"
	m.Files[xdg] = []byte("x")
	m.Results["git config --file "+xdg+" --name-only --list"] = effecttest.Result{Out: []byte("user.email\n")}
	m.Results["git config --file "+gitConfigFile+" --type path --get-all include.path"] = effecttest.Result{Err: exitOne(t)}
	p := mustPlan(t, g)
	if last := p.Changes[len(p.Changes)-1].String(); !strings.Contains(last, "settings of user.email come before the include") {
		t.Errorf("last change = %q; want the warning of user.email", last)
	}
	if len(out.Lines) != 0 {
		t.Errorf("Plan said %q; want the warning once the changes are made", out.Lines)
	}
}
