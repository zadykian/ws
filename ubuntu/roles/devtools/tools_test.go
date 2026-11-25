package devtools_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/ubuntu/roles/devtools"
)

// version is the version that [tools] pins for the tool.
func (c miseConfig) version(t *testing.T, tool string) string {
	switch v := c.Tools[tool].(type) {
	case string:
		return v
	case map[string]any:
		if s, ok := v["version"].(string); ok {
			return s
		}
	}
	t.Fatalf("[tools] pins no version of %s: %v", tool, c.Tools[tool])
	return ""
}

// unitsFile is what the tests read of units.toml.tmpl. Its template's marks sit inside strings, so it
// reads as TOML before a render.
type unitsFile struct {
	File map[string]struct {
		Path  string `toml:"path"`
		Link  string `toml:"link"`
		After string `toml:"after"`
	} `toml:"file"`
}

func readUnits(t *testing.T) unitsFile {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	var u unitsFile
	if err := toml.Unmarshal(text, &u); err != nil {
		t.Fatal(err)
	}
	return u
}

// installs is where mise puts a tool, under root's data directory, which mise takes by default.
const installs = "/root/.local/share/mise/installs/"

// links names each link unit's tool in [tools], and the path of the program below its version's
// directory.
var links = map[string]struct{ tool, below string }{
	"devtools-go":        {"go", ""},
	"devtools-gopls":     {"go:golang.org/x/tools/gopls", "/bin/gopls"},
	"devtools-dlv":       {"go:github.com/go-delve/delve/cmd/dlv", "/bin/dlv"},
	"devtools-pgls":      {"github:supabase-community/postgres-language-server", "/postgres-language-server"},
	"devtools-docker-ls": {"github:docker/docker-language-server", "/docker-language-server"},
}

// A link names its tool's version, as mise names the directory: the tool's name with each ':', '/'
// and '.' made '-'. A bump of [tools] alone fails here.
func TestDevtoolsLinksNameThePinnedVersions(t *testing.T) {
	c, u := readConfig(t), readUnits(t)
	tools := 0
	for _, f := range u.File {
		if strings.HasPrefix(f.Link, installs) {
			tools++
		}
	}
	if tools != len(links) {
		t.Errorf("%d links into mise's tools, want %d", tools, len(links))
	}
	dir := strings.NewReplacer(":", "-", "/", "-", ".", "-")
	for name, l := range links {
		f, ok := u.File[name]
		if !ok {
			t.Errorf("no unit file.%s", name)
			continue
		}
		if want := installs + dir.Replace(l.tool) + "/" + c.version(t, l.tool) + l.below; f.Link != want {
			t.Errorf("%s links to %q, want %q", name, f.Link, want)
		}
		if f.After != "" {
			t.Errorf("%s waits on %q, want nothing", name, f.After)
		}
		if c.Bootstrap.Packages["ws:file/"+name] != "in-sync" {
			t.Errorf("mise.guest.toml lacks ws:file/%s", name)
		}
	}
}

// The editors name the servers by the links' paths.
func TestServerPathsAreTheLinks(t *testing.T) {
	u, paths := readUnits(t), devtools.ServerPaths("/home/u")
	for server, unit := range map[string]string{
		"gopls":                    "devtools-gopls",
		"postgres-language-server": "devtools-pgls",
		"docker-language-server":   "devtools-docker-ls",
	} {
		if got := u.File[unit].Path; got != paths[server] {
			t.Errorf("%s is at %q, but ServerPaths names %q", unit, got, paths[server])
		}
	}
	if u.File["devtools-dlv"].Path != devtools.DlvPath || u.File["devtools-go"].Path != "/usr/local/go" {
		t.Errorf("dlv at %q and go at %q, want %s and /usr/local/go", u.File["devtools-dlv"].Path,
			u.File["devtools-go"].Path, devtools.DlvPath)
	}
}

// lockEntry is a tool's entry in mise's lock: its version, and the platforms' tables beside it.
type lockEntry map[string]any

// The lock holds each tool at its version, under the tree's locked = true. A download's entry holds its url
// and SHA-256 for both platforms; the servers' sums are those ws pinned in Go before mise. The lock
// holds other roles' tools too, so the server's TestServerLockHoldsEachPinnedTool counts them.
func TestDevtoolsLockHoldsEachTool(t *testing.T) {
	text, err := os.ReadFile("../../guest/mise/mise.guest.lock")
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Tools map[string][]lockEntry `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &lock); err != nil {
		t.Fatal(err)
	}
	c := readConfig(t)
	sums := map[string]map[string]string{
		"go": {},
		"github:supabase-community/postgres-language-server": {
			"linux-x64":   "82ff82e39be65824378afa5d02626a1b42e8d8e48f97cb4938e4d477c8a9be52",
			"linux-arm64": "7be7de3660df047fd712f8fca4d6c42ecf1515daad952715eb02e29053ce256c",
		},
		"github:docker/docker-language-server": {
			"linux-x64":   "01907aa5b0eae11e44cffea0a993d08aa155542a9af570295dd1dff39e67692a",
			"linux-arm64": "bd56c7815e0a22cfb708669f3d5e817de91d9b54039ff7e52867142a132ad8d7",
		},
	}
	sha := regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	for tool := range c.Tools {
		entries := lock.Tools[tool]
		if len(entries) != 1 || entries[0]["version"] != c.version(t, tool) {
			t.Errorf("%s: the lock holds %v, want one entry at %s", tool, entries, c.version(t, tool))
			continue
		}
		want, download := sums[tool]
		for _, platform := range []string{"linux-x64", "linux-arm64"} {
			p, _ := entries[0]["platforms."+platform].(map[string]any)
			switch {
			case !download && p != nil:
				t.Errorf("%s: %s has a download, want the version alone", tool, platform)
			case !download:
			case p == nil || !sha.MatchString(fmt.Sprint(p["checksum"])) || p["url"] == nil:
				t.Errorf("%s: %s has %v, want a url and a sha256", tool, platform, p)
			case want[platform] != "" && p["checksum"] != "sha256:"+want[platform]:
				t.Errorf("%s: %s's checksum %v, want %s", tool, platform, p["checksum"], want[platform])
			}
		}
	}
}
