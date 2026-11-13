package devtools

import (
	"strings"
	"testing"
)

func TestRepositoriesTable(t *testing.T) {
	for _, repo := range repositories("resolute") {
		if len(repo.Fingerprints) == 0 || repo.Key == "" || repo.URIs == "" {
			t.Errorf("%s: %+v", repo.Name, repo)
		}
		for _, p := range repo.Fingerprints {
			if len(p) != 40 {
				t.Errorf("%s: the fingerprint %q", repo.Name, p)
			}
		}
	}
	if got := repositories("resolute")[2]; got.Suites != "resolute" || !got.Needed {
		t.Errorf("the PPA is %+v", got)
	}
	if got := repositories("resolute")[3]; got.Name != "github-cli" || len(got.Fingerprints) != 2 || !got.Needed {
		t.Errorf("GitHub CLI's repository is %+v", got)
	}
}

func TestGHUpgradesAllowTheRepository(t *testing.T) {
	gh := repositories("resolute")[3]
	host := strings.TrimPrefix(strings.TrimSuffix(gh.URIs, "/packages"), "https://")
	want := `"origin=gh,codename=` + gh.Suites + `,site=` + host + `";`
	if !strings.Contains(string(ghUpgrades), want) {
		t.Errorf("52unattended-upgrades-github-cli lacks %s", want)
	}
}
