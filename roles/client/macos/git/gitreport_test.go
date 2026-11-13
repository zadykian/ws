package git_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
)

func TestGitWithoutNameReportsNothing(t *testing.T) {
	_, g := gitReady()
	g.Conf = conf.Conf{}
	if got := mactest.ReportOf(t, g.Step()); len(got) != 0 {
		t.Errorf("items = %v, want none", got)
	}
}

func TestGitReportsWhatItManages(t *testing.T) {
	m, g := gitReady()
	dir := gitHome + "/.config/ws/"
	m.Files[dir+"gitconfig"] = []byte("[user]\n    name = other\n")
	got := mactest.ReportOf(t, g.Step())
	if it := got["gh"]; it.Kind != step.PackageItem || it.Found != "installed" {
		t.Errorf("gh = %+v", it)
	}
	if it := got[dir+"allowed_signers"]; it.Want != "mode 644" || it.Found != "missing" {
		t.Errorf("allowed_signers = %+v", it)
	}
	if it := got[dir+"gitconfig"]; !strings.Contains(it.Conflict, "no Written by line") || it.Diff == "" {
		t.Errorf("gitconfig = %+v, want a file ws did not write, with its diff", it)
	}
	if it := got[gitHome+"/.config"]; it.Kind != step.DirItem || it.Found != "present" {
		t.Errorf("~/.config = %+v", it)
	}
	if it := got[gitConfigFile]; it.Want != it.Found || it.Diff != "" {
		t.Errorf("~/.gitconfig = %+v, want its include held, and no diff", it)
	}
	if it := got[gitPub]; it.Kind != step.FileItem || it.Found != "present" {
		t.Errorf("key = %+v, want its .pub found", it)
	}
	if slices.Contains(m.Calls, "read "+strings.TrimSuffix(gitPub, ".pub")) {
		t.Errorf("calls = %q, which read the private key", m.Calls)
	}
}

func TestGitReportShowsAnIncludeWhereThePlanStops(t *testing.T) {
	m, g := gitReady()
	gitConfig(m, "x", "", "user.name\n")
	m.Links[gitConfigFile] = "/dotfiles/gitconfig"
	m.Files["/dotfiles/gitconfig"] = []byte("x")
	items, err := g.Step().Report(context.Background())
	if err != nil || !slices.ContainsFunc(items, func(it step.Item) bool {
		return it.Name == gitConfigFile && strings.Contains(it.Conflict, "a link to /dotfiles/gitconfig")
	}) {
		t.Errorf("Report = %+v, %v; want ~/.gitconfig as a link in conflict", items, err)
	}
}

func TestGitReportShowsGhToInstall(t *testing.T) {
	m, g := gitReady()
	g.LookPath = func(name string) (string, error) {
		if name == "gh" {
			return "", fmt.Errorf("no %s", name)
		}
		return "/opt/bin/" + name, nil
	}
	m.Results["brew --prefix"] = effecttest.Result{Out: []byte("/opt\n")}
	got := mactest.ReportOf(t, g.Step())
	if got["gh"].Found != "missing" || got[gitPub].Found != "present" {
		t.Errorf("gh = %+v, key = %+v, want gh missing", got["gh"], got[gitPub])
	}
}

// A report shows ws's lines alone of the user's ~/.gitconfig.
func TestGitReportShowsWsLinesAloneOfAUsersFile(t *testing.T) {
	m, _, g := gitWithConfig(mactest.UsersText, "", "")
	m.Results["git config --file "+gitConfigFile+" --type path --get-all include.path"] =
		effecttest.Result{Err: mactest.ExitOne(t)}
	mactest.CheckUsersFile(t, g.Step(), "+[include]")
}

func TestGitReportWarnsNothing(t *testing.T) {
	m, out, g := gitWithConfig("[user]\n    name = X\n", "", "user.name\n")
	m.Results["git config --file "+gitConfigFile+" --type path --get-all include.path"] =
		effecttest.Result{Err: mactest.ExitOne(t)}
	if _, err := g.Step().Report(context.Background()); err != nil || len(out.Lines) != 0 {
		t.Errorf("Report = %v, lines %q; want no warning", err, out.Lines)
	}
}
