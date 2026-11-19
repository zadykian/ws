package lazygit_test

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/values"
)

const (
	home       = "/home/zadykian"
	configPath = home + "/.config/lazygit/config.yml"
)

// configUnit loads the role's unit on a machine with the account zadykian, 1000:1000, and its home.
func configUnit(t *testing.T) (*effecttest.Machine, kind.Unit, []byte) {
	return configUnitOf(t, "zadykian", "1000", home)
}

// configUnitOf loads the role's unit for WS_USER user, whose uid and gid are id, at home dir.
func configUnitOf(t *testing.T, user, id, dir string) (*effecttest.Machine, kind.Unit, []byte) {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/config.yml"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["lazygit/"+name] = &fstest.MapFile{Data: text}
	}
	v, err := values.Parse([]byte("WS_USER: " + user + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["id -u "+user] = effecttest.Result{Out: []byte(id + "\n")}
	m.Results["id -g "+user] = effecttest.Result{Out: []byte(id + "\n")}
	m.Results["getent passwd "+user] = effecttest.Result{Out: []byte(user + ":x:" + id + ":" + id + "::" + dir + ":/bin/bash\n")}
	m.Dirs[dir] = 0o750
	e := kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v}
	us, err := kind.Load(e, fsys)
	if err != nil || len(us) != 1 || us[0].Name != "file/lazygit-config" {
		t.Fatalf("units = %v, %v; want file/lazygit-config alone", kind.Names(us), err)
	}
	return m, us[0], fsys["lazygit/files/config.yml"].Data
}

func write(t *testing.T, u kind.Unit) error {
	_, err := step.Run(t.Context(), kind.Steps([]kind.Unit{u}), &effecttest.Out{}, false)
	return err
}

// The user owns the file, and a second write changes nothing. A config of the user's own stays
// beside the new one.
func TestLazygitConfigIsTheUsersAndKeepsACopy(t *testing.T) {
	m, u, config := configUnit(t)
	m.Dirs[home+"/.config"], m.Owners[home+"/.config"] = 0o755, [2]int{1000, 1000}
	m.Dirs[home+"/.config/lazygit"], m.Owners[home+"/.config/lazygit"] = 0o755, [2]int{1000, 1000}
	m.Files[configPath], m.Owners[configPath] = []byte("by hand\n"), [2]int{1000, 1000}
	if err := write(t, u); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(m.Files[configPath], config) || m.Owners[configPath] != [2]int{1000, 1000} {
		t.Errorf("config = %d bytes, owner %v; want files/config.yml, the user's", len(m.Files[configPath]), m.Owners[configPath])
	}
	var copies int
	for p, data := range m.Files {
		if strings.HasPrefix(p, configPath+".") && strings.HasSuffix(p, "~") && string(data) == "by hand\n" {
			copies++
		}
	}
	if copies != 1 {
		t.Errorf("%d copies of the old config, want 1", copies)
	}
	if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second plan = %v, %v; want none", p.Changes, err)
	}
}

func TestLazygitConfigMakesItsDirectory(t *testing.T) {
	m, u, _ := configUnit(t)
	if err := write(t, u); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{home + "/.config", home + "/.config/lazygit"} {
		if _, ok := m.Dirs[d]; !ok || m.Owners[d] != [2]int{1000, 1000} {
			t.Errorf("%s = %v, owner %v; want the user's directory", d, m.Dirs[d], m.Owners[d])
		}
	}
	if len(m.Files) != 1 {
		t.Errorf("files %d, want the config alone, with no copy of a file that was not there", len(m.Files))
	}
}

// A link on the way is refused, in root's home too, where WS_USER is root on a cloud VM.
func TestLazygitConfigRefusesALink(t *testing.T) {
	for _, c := range []struct{ user, id, dir string }{{"zadykian", "1000", home}, {"root", "0", "/root"}} {
		m, u, _ := configUnitOf(t, c.user, c.id, c.dir)
		m.Links[c.dir+"/.config"] = "/srv/target"
		if err := write(t, u); err == nil || !strings.Contains(err.Error(), c.dir+"/.config is a link to /srv/target") {
			t.Errorf("%s: write = %v, want the link refused", c.user, err)
		}
		if len(m.Files) != 0 {
			t.Errorf("%s: files %v, want none", c.user, m.Files)
		}
	}
}
