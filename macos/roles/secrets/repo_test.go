package secrets_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

func TestParseRepoTakesGitHubsSSHURL(t *testing.T) {
	for _, url := range []string{"git@github.com:o/n.git", "git@github.com:o/n"} {
		r, err := secrets.ParseRepo(url)
		if err != nil {
			t.Fatalf("%s: %v", url, err)
		}
		if r.Slug() != "o/n" || r.HTTPS() != "https://github.com/o/n.git" {
			t.Errorf("%s gives %s and %s, want o/n and its HTTPS URL", url, r.Slug(), r.HTTPS())
		}
	}
	for _, url := range []string{
		"https://github.com/o/n.git",
		"git@gitlab.com:o/n.git",
		"git@github.com:o/n/x.git",
		"git@github.com:o/.git",
		"git@github.com:o",
		"git@github.com:/n.git",
		"git@github.com:o/n m.git",
		"git@github.com:o/...git",
	} {
		if r, err := secrets.ParseRepo(url); err == nil {
			t.Errorf("%s gives %s, want an error", url, r.Slug())
		}
	}
}

func TestRepoOfReadsMiseTomlsEnv(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/opt/ws/mise.toml"] = []byte("[settings]\nuse_versions_host = false\n\n" +
		"[env]\nWS_SECRETS_REPO = \"git@github.com:o/n.git\"\n")
	r, err := secrets.RepoOf(m, secrets.Tree)
	if err != nil {
		t.Fatal(err)
	}
	if r != (secrets.Repo{Owner: "o", Name: "n"}) {
		t.Errorf("RepoOf gives %+v, want o/n", r)
	}

	m.Files["/opt/ws/mise.toml"] = []byte("[env]\nOTHER = \"x\"\n")
	_, err = secrets.RepoOf(m, secrets.Tree)
	if err == nil || !strings.Contains(err.Error(), "/opt/ws/mise.toml") ||
		!strings.Contains(err.Error(), secrets.RepoKey) {
		t.Errorf("a mise.toml without the key gives %v, want an error naming the file and %s", err, secrets.RepoKey)
	}

	m.Files["/opt/ws/mise.toml"] = []byte("[env]\nWS_SECRETS_REPO = \"https://github.com/o/n.git\"\n")
	if _, err := secrets.RepoOf(m, secrets.Tree); err == nil || !strings.Contains(err.Error(), secrets.RepoKey) {
		t.Errorf("an HTTPS URL gives %v, want an error naming %s", err, secrets.RepoKey)
	}
}
