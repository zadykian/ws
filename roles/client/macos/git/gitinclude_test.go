package git_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
)

func TestGitWarnsOfOwnSettingsBelowTheInclude(t *testing.T) {
	m, out, g := gitWithConfig("[user]\n    name = X\n", "", "user.name\n")
	m.Results["git config --file "+gitConfigFile+" --type path --get-all include.path"] =
		effecttest.Result{Err: mactest.ExitOne(t)}
	m.Results["cp -p "+gitConfigFile+" "+gitConfigFile+".20260102030405~"] = effecttest.Result{}
	if err := g.Apply(context.Background(), mustPlan(t, g)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "own settings of user.name, below the include") {
		t.Errorf("no warning of user.name: %q", out.Lines)
	}
}

func TestGitWarnsOfXDGSettingsThatLose(t *testing.T) {
	m, out, g := gitWithConfig("x", "", "user.name\n")
	xdg := gitHome + "/.config/git/config"
	m.Files[xdg] = []byte("x")
	m.Results["git config --file "+xdg+" --name-only --list"] = effecttest.Result{Out: []byte("user.email\n")}
	m.Results["git config --file "+gitConfigFile+" --type path --get-all include.path"] = effecttest.Result{Err: mactest.ExitOne(t)}
	p := mustPlan(t, g)
	if last := p.Changes[len(p.Changes)-1].String(); !strings.Contains(last, "settings of user.email come before the include") {
		t.Errorf("last change = %q; want the warning of user.email", last)
	}
	if len(out.Lines) != 0 {
		t.Errorf("Plan said %q; want the warning once the changes are made", out.Lines)
	}
}
