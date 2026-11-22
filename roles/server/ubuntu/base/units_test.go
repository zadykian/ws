package base_test

import (
	"os"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// gitCorePrint is Launchpad's fingerprint of the git-core PPA's signing key.
const gitCorePrint = "F911AB184317630C59970973E363C90F8F1B6217"

// A git below 2.54 lacks the hooks that git's config holds, so git comes from the PPA, whose key
// the source pins by the fingerprint that Launchpad lists.
func TestGitComesFromThePPAByItsPinnedKey(t *testing.T) {
	text, err := os.ReadFile("mise.server.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Apt struct {
				Sources map[string]struct {
					PPA         string `toml:"ppa"`
					Fingerprint string `toml:"fingerprint"`
				} `toml:"sources"`
			} `toml:"apt"`
			Packages map[string]string `toml:"packages"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	src, ok := c.Bootstrap.Apt.Sources["git-core"]
	if !ok {
		t.Fatal("no source git-core")
	}
	if src.PPA != "git-core/ppa" {
		t.Errorf("ppa %q, want git-core/ppa", src.PPA)
	}
	if got := strings.ReplaceAll(src.Fingerprint, " ", ""); got != gitCorePrint {
		t.Errorf("fingerprint %q, want %s", src.Fingerprint, gitCorePrint)
	}
	if c.Bootstrap.Packages["apt:git"] != "latest" {
		t.Errorf("apt:git is %q, want latest", c.Bootstrap.Packages["apt:git"])
	}
}

// Unattended upgrades take a package from an origin that their config allows. The PPA's git is
// above Ubuntu's, so without the PPA's origin git would stay at its first install.
func TestUpgradesAllowThePPAsOrigin(t *testing.T) {
	text, err := os.ReadFile("mise.server.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Files map[string]struct {
				Source string `toml:"source"`
			} `toml:"files"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	const dest = "/etc/apt/apt.conf.d/52unattended-upgrades-git-core"
	f, ok := c.Bootstrap.Files[dest]
	if !ok {
		t.Fatalf("no file %s", dest)
	}
	conf, err := os.ReadFile(f.Source)
	if err != nil {
		t.Fatal(err)
	}
	want := `"origin=LP-PPA-git-core,codename=${distro_codename}";`
	if !strings.Contains(string(conf), want) {
		t.Errorf("%s lacks %s", f.Source, want)
	}
}
