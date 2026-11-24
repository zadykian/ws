package signoz_test

import (
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/ubuntu/roles/signoz"
)

// miseFiles are the mise files that hold the forged files: the mode's, which a host loads too.
var miseFiles = []string{"mise.self-contained.toml"}

// miseFile is what the test reads of an entry of a mise file.
type miseFile struct {
	Source string `toml:"source"`
	Owner  string `toml:"owner"`
	Group  string `toml:"group"`
	Mode   string `toml:"mode"`
}

// TestEachForgedFileIsMises holds foundryctl's files and mise's entries together, as mise copies
// no tree: a file that a new forge adds has its entry, and an entry its file.
func TestEachForgedFileIsMises(t *testing.T) {
	for _, file := range miseFiles {
		t.Run(file, func(t *testing.T) { testEachForgedFileIsMises(t, file) })
	}
}

// testEachForgedFileIsMises is TestEachForgedFileIsMises for the mise file named file.
func testEachForgedFileIsMises(t *testing.T, file string) {
	text, err := os.ReadFile(file) //nolint:gosec // a mise file of the role, named by the test
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Files       map[string]miseFile          `toml:"files"`
			Directories map[string]map[string]string `toml:"directories"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	files, dirs := map[string]bool{}, map[string]bool{}
	err = fs.WalkDir(os.DirFS("deployment"), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		to := path.Join(signoz.Dir, rel)
		if d.IsDir() {
			dirs[to] = true
			if c.Bootstrap.Directories[to]["mode"] != "0755" {
				t.Errorf("%s: want a directory entry with mode 0755", to)
			}
			return nil
		}
		files[to] = true
		want := miseFile{Source: "deployment/" + rel, Owner: "root", Group: "root", Mode: "0644"}
		if got := c.Bootstrap.Files[to]; got != want {
			t.Errorf("%s: entry %+v, want %+v", to, got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for to := range c.Bootstrap.Files {
		if !files[to] {
			t.Errorf("%s: an entry with no forged file", to)
		}
	}
	for to := range c.Bootstrap.Directories {
		if !dirs[to] {
			t.Errorf("%s: an entry with no forged directory", to)
		}
	}
}
