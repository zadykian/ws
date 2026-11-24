package devtools_test

import (
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// source is what the tests read of an apt source in mise.guest.toml.
type source struct {
	URIs        string `toml:"uris"`
	Suites      string `toml:"suites"`
	Components  string `toml:"components"`
	Key         string `toml:"key"`
	PPA         string `toml:"ppa"`
	Fingerprint any    `toml:"fingerprint"`
}

// fingerprints returns the source's fingerprints, a string or a list.
func (s source) fingerprints() []string {
	switch f := s.Fingerprint.(type) {
	case string:
		return []string{f}
	case []any:
		var out []string
		for _, v := range f {
			out = append(out, v.(string)) //nolint:forcetypeassert // a test of TOML read as strings
		}
		return out
	}
	return nil
}

type miseConfig struct {
	Bootstrap struct {
		Apt struct {
			Sources map[string]source `toml:"sources"`
		} `toml:"apt"`
		Packages map[string]string `toml:"packages"`
	} `toml:"bootstrap"`
	// Tools holds a version, or a table with one.
	Tools map[string]any `toml:"tools"`
}

func readConfig(t *testing.T) miseConfig {
	text, err := os.ReadFile("mise.guest.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c miseConfig
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDevtoolsSourcesPinTheirKeys(t *testing.T) {
	want := map[string]source{
		"nodesource": {URIs: "https://deb.nodesource.com/node_24.x", Suites: "nodistro", Components: "main",
			Key: "https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key"},
		"charm": {URIs: "https://repo.charm.sh/apt/", Suites: "*", Components: "*",
			Key: "https://repo.charm.sh/apt/gpg.key"},
		"github-cli": {URIs: "https://cli.github.com/packages", Suites: "stable", Components: "main",
			Key: "https://cli.github.com/packages/githubcli-archive-keyring.asc"},
	}
	prints := map[string][]string{
		"nodesource": {"6F71F525282841EEDAF851B42F59B5F99B1BE0B4"},
		"charm":      {"ED927B38BE981E53CA09153D03BBF595D4DFD35C"},
		"github-cli": {"2C6106201985B60E6C7AC87323F3D4EA75716059", "7F38BBB59D064DBCB3D84D725612B36462313325"},
	}
	got := readConfig(t).Bootstrap.Apt.Sources
	if len(got) != len(want) {
		t.Errorf("%d sources, want %d", len(got), len(want))
	}
	hex := regexp.MustCompile(`^[0-9A-F]{40}$`)
	for name, w := range want {
		g := got[name]
		fp := g.fingerprints()
		g.Fingerprint, w.Fingerprint = nil, nil
		if g != w {
			t.Errorf("%s: %+v, want %+v", name, g, w)
		}
		if !slices.Equal(fp, prints[name]) {
			t.Errorf("%s: fingerprints %v, want %v", name, fp, prints[name])
		}
		for _, f := range fp {
			if !hex.MatchString(f) {
				t.Errorf("%s: the fingerprint %q is not 40 hex digits", name, f)
			}
		}
	}
}

// The unattended-upgrades pattern names the origin, the suite and the host of gh's source.
func TestGHUpgradesAllowTheRepository(t *testing.T) {
	gh := readConfig(t).Bootstrap.Apt.Sources["github-cli"]
	u, err := url.Parse(gh.URIs)
	if err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile("files/52unattended-upgrades-github-cli")
	if err != nil {
		t.Fatal(err)
	}
	want := `"origin=gh,codename=` + gh.Suites + `,site=` + u.Host + `";`
	if !strings.Contains(string(text), want) {
		t.Errorf("52unattended-upgrades-github-cli lacks %s", want)
	}
}

func TestDevtoolsPackagesAreTodays(t *testing.T) {
	want := []string{"apt:nodejs", "apt:gh", "apt:glow", "apt:libicu78",
		"apt:libasound2t64", "apt:libatk-bridge2.0-0t64", "apt:libatk1.0-0t64", "apt:libcairo2",
		"apt:libcups2t64", "apt:libgbm1", "apt:libnspr4", "apt:libnss3", "apt:libpango-1.0-0",
		"apt:libxcomposite1", "apt:libxdamage1", "apt:libxfixes3", "apt:libxkbcommon0", "apt:libxrandr2"}
	got := readConfig(t).Bootstrap.Packages
	apt := 0
	for name := range got {
		if strings.HasPrefix(name, "apt:") {
			apt++
		}
	}
	if apt != len(want) {
		t.Errorf("%d apt packages, want %d", apt, len(want))
	}
	for _, name := range want {
		if got[name] != "latest" {
			t.Errorf("%s = %q, want latest", name, got[name])
		}
	}
}
