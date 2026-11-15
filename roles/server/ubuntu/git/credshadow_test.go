package git

import (
	"context"
	"strings"
	"testing"
)

// ghSections is what gh auth setup-git appends with --global, below the include, as git prints it.
const ghSections = "credential.https://github.com.helper \n" +
	"credential.https://github.com.helper !/usr/bin/gh auth git-credential\n" +
	"credential.https://gist.github.com.helper \n" +
	"credential.https://gist.github.com.helper !/usr/bin/gh auth git-credential\n"

func shadowed(t *testing.T, rc, helpers string, creds ...string) string {
	t.Helper()
	m, sm, out, g := gitFixture()
	for _, u := range creds {
		g.Creds = append(g.Creds, Credential{URL: u})
	}
	m.Files["/home/ws/.gitconfig"] = []byte(rc)
	sm.helpers[rc[:8]] = helpers
	sm.paths[rc[:8]] = "/etc/ws/gitconfig\n"
	sm.answers[rc[:8]] = "include.path\ncredential.https://github.com.helper\n"
	if _, err := g.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	return strings.Join(out.Lines, "\n")
}

const included = "[include]\n\tpath = /etc/ws/gitconfig\n"

func TestGitconfigWarnsOfGhSectionsThatResetATokenOnEachRun(t *testing.T) {
	rc := included + "[credential \"https://github.com\"]\n\thelper = \n\thelper = !/usr/bin/gh auth git-credential\n"
	warned := shadowed(t, rc, ghSections, "https://github.com/org")
	for _, want := range []string{`[credential "https://github.com"]`, "/etc/ws/tokens.gitconfig", "remove them"} {
		if !strings.Contains(warned, want) {
			t.Errorf("warning %q lacks %q", warned, want)
		}
	}
	if strings.Contains(warned, "gist") || strings.Contains(warned, "!/usr/bin/gh") {
		t.Errorf("warning %q names a section that holds no token, or a value", warned)
	}
}

func TestGitconfigWarnsOfNoCredentialSection(t *testing.T) {
	warned := shadowed(t, included+"[user]\n\tname = x\n", "", "https://github.com/org")
	if strings.Contains(warned, "credential") {
		t.Errorf("said %q", warned)
	}
}

func TestGitconfigWarnsOfASectionThatShadowsNoToken(t *testing.T) {
	warned := shadowed(t, included+"[credential \"https://github.com\"]\n", ghSections, "https://git.example.com")
	if strings.Contains(warned, "credential") {
		t.Errorf("said %q", warned)
	}
}

func TestGitconfigWarnsOfNoSectionWhereNoTokenExists(t *testing.T) {
	warned := shadowed(t, included+"[credential \"https://github.com\"]\n", ghSections)
	if strings.Contains(warned, "credential") {
		t.Errorf("said %q", warned)
	}
}

func TestGitconfigWarnsOfATokenThatTheFileHoldsFromAnEarlierRun(t *testing.T) {
	m, sm, out, g := gitFixture()
	m.Files["/etc/ws/tokens.gitconfig"] = []byte("x")
	m.Files["/home/ws/.gitconfig"] = []byte(included + "[credential]\n")
	sm.answers["x"] = "credential.https://github.com/org.helper\n"
	sm.answers[included[:8]] = "include.path\ncredential.helper\n"
	sm.helpers[included[:8]] = "credential.helper \n"
	sm.paths[included[:8]] = "/etc/ws/gitconfig\n"
	if _, err := g.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if all := strings.Join(out.Lines, "\n"); !strings.Contains(all, "[credential]") {
		t.Errorf("said %q", all)
	}
}

func TestGitconfigNamesGitsXDGFileSettingsWhereItAddsTheInclude(t *testing.T) {
	m, sm, out, g := gitFixture()
	m.Files["/home/ws/.gitconfig"] = []byte("[core]\n\teditor = vi\n")
	m.Files["/home/ws/.config/git/config"] = []byte("[user]\n\temail = other@x\n")
	sm.answers["[core]"] = "core.editor\n"
	sm.answers["[user]"] = "user.email\n"
	if _, err := g.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "/home/ws/.config/git/config's settings of user.email come before the include in ~/.gitconfig, " +
		"so /etc/ws/gitconfig's win over them"
	if all := strings.Join(out.Lines, "\n"); !strings.Contains(all, want) || strings.Contains(all, "other@x") {
		t.Errorf("said %q", all)
	}
}
