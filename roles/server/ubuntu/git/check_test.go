package git

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/server/values"
)

// checked runs Check on the fixture's machine with the values v.
func checked(t *testing.T, sm stdinMachine, v values.Values) ([]string, error) {
	_, _, g := gitFixture()
	return Check(t.Context(), sm, g.A, v, nil)
}

func TestCheckWithoutTheTokensFile(t *testing.T) {
	_, sm, _ := gitFixture()
	notes, err := checked(t, sm, values.Values{})
	if err != nil || notes != nil {
		t.Errorf("Check = %q, %v; want nothing", notes, err)
	}
}

func TestCheckRefusesABadToken(t *testing.T) {
	m, sm, _ := gitFixture()
	m.Files[TokensFile] = []byte(`["https://git.example.test/a b"]` + "\ntoken = \"hunter2\"\n")
	_, err := checked(t, sm, values.Values{})
	if err == nil || !strings.Contains(err.Error(), TokensFile+": each table") || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("Check = %v; want checkCredentials' refusal, with no token in it", err)
	}
}

func TestCheckRefusesABadClone(t *testing.T) {
	_, sm, _ := gitFixture()
	repo := "https://me:hunter2@example.test/r.git" //nolint:gosec // G101: a fake token, which the check refuses
	v := values.Values{Git: values.Git{Clones: []values.GitClone{{Repo: repo, Dest: "r"}}}}
	if _, err := checked(t, sm, v); err == nil || !strings.Contains(err.Error(), "WS_GIT_CLONES") ||
		strings.Contains(err.Error(), "hunter2") {
		t.Errorf("Check = %v; want checkClones' refusal, with no repo in it", err)
	}
}

func TestCheckNamesTheUsersOwnSettingsWithoutTheInclude(t *testing.T) {
	m, sm, _ := gitFixture()
	m.Files["/home/ws/.gitconfig"] = []byte("[user]\n\temail = a@b.c\n[http]\n\textraheader = hunter2\n")
	sm.answers["[user]"] = "user.email\nhttp.extraheader\n"
	notes, err := checked(t, sm, values.Values{})
	if all := strings.Join(notes, "\n"); err != nil || !strings.Contains(all, "own settings of user.email, below the include") ||
		strings.Contains(all, "hunter2") {
		t.Errorf("Check = %q, %v", notes, err)
	}
}

func TestCheckNamesNoOwnSettingsBelowTheInclude(t *testing.T) {
	m, sm, _ := gitFixture()
	m.Files["/home/ws/.gitconfig"] = []byte(included + "[user]\n\temail = a@b.c\n")
	sm.answers["[include]"] = "include.path\nuser.email\n"
	sm.paths["[include]"] = "/etc/ws/gitconfig\n"
	if notes, err := checked(t, sm, values.Values{}); err != nil || notes != nil {
		t.Errorf("Check = %q, %v; want no note once the include is there", notes, err)
	}
}

func TestCheckLeavesAMisplacedIncludeToItsUnit(t *testing.T) {
	m, sm, _ := gitFixture()
	m.Files["/home/ws/.gitconfig"] = []byte("[user]\n\tname = x\n" + included)
	sm.answers["[user]"] = "user.name\ninclude.path\n"
	sm.paths["[user]"] = "/etc/ws/gitconfig\n"
	if notes, err := checked(t, sm, values.Values{}); err != nil || notes != nil {
		t.Errorf("Check = %q, %v; want neither a note nor an error", notes, err)
	}
}
