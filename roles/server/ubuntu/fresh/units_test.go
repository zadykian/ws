package fresh_test

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// load builds a machine with the account's home, its ids and the architecture, and loads the role's
// units for the user and home.
func load(t *testing.T, user, home, arch string) (*effecttest.Machine, map[string]kind.Unit) {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml", "files/config.json.tmpl", "files/themes/jetbrains-dark.json"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["fresh/"+name] = &fstest.MapFile{Data: text}
	}
	v, err := values.Parse([]byte("WS_USER: " + user + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	uid := "1000"
	if user == "root" {
		uid = "0"
	}
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte(arch + "\n")}
	m.Results["id -u "+user] = effecttest.Result{Out: []byte(uid + "\n")}
	m.Results["id -g "+user] = effecttest.Result{Out: []byte(uid + "\n")}
	m.Results["getent passwd "+user] = effecttest.Result{Out: []byte(user + ":x:" + uid + ":" + uid + "::" + home + ":/bin/bash\n")}
	m.Dirs[home] = 0o750
	m.Packages = effecttest.Packages{}
	e := kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v}
	us, err := kind.Load(e, fsys)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]kind.Unit{}
	for _, u := range us {
		byName[u.Name] = u
	}
	if len(byName) != 3 || byName["deb/fresh"].Name == "" || byName["file/fresh-config"].Name == "" || byName["file/fresh-theme"].Name == "" {
		t.Fatalf("units = %v; want deb/fresh, file/fresh-config and file/fresh-theme", kind.Names(us))
	}
	return m, byName
}

func run(t *testing.T, us map[string]kind.Unit, names ...string) error {
	var list []kind.Unit
	for _, n := range names {
		list = append(list, us[n])
	}
	_, err := step.Run(t.Context(), kind.Steps(list), &effecttest.Out{}, false)
	return err
}

// The .deb is the pin: the release's file for the architecture, checked by its sum.
func TestFreshDebIsThePin(t *testing.T) {
	_, us := load(t, "zadykian", "/home/zadykian", "arm64")
	p, err := us["deb/fresh"].Step.Plan(t.Context())
	if err != nil || len(p.Changes) != 3 || p.Changes[2].String() != "install fresh-editor 0.5.7-1" {
		t.Fatalf("plan = %v, %v; want the directory, the fetch and the install", p.Changes, err)
	}
	want := "https://github.com/sinelaw/fresh/releases/download/v0.5.7/fresh-editor_0.5.7-1_arm64.deb"
	if got := p.Changes[1].String(); !strings.Contains(got, want) {
		t.Errorf("fetch = %q, want it to name %s", got, want)
	}
}

func TestFreshDebFailsWithNoSum(t *testing.T) {
	_, us := load(t, "zadykian", "/home/zadykian", "riscv64")
	if _, err := us["deb/fresh"].Step.Plan(t.Context()); err == nil {
		t.Error("no error")
	}
}

// The servers sit under the user's home: a literal home would point four servers nowhere.
func TestFreshConfigNamesTheHomesServers(t *testing.T) {
	for user, home := range map[string]string{"zadykian": "/home/zadykian", "root": "/root"} {
		m, us := load(t, user, home, "amd64")
		if err := run(t, us, "file/fresh-config", "file/fresh-theme"); err != nil {
			t.Fatal(err)
		}
		path := home + "/.config/fresh/config.json"
		if got := string(m.Files[path]); !strings.Contains(got, `"`+home+`/go/bin/gopls"`) {
			t.Errorf("%s: config.json lacks the home's gopls", user)
		}
		if _, ok := m.Files[home+"/.config/fresh/themes/jetbrains-dark.json"]; !ok {
			t.Errorf("%s: no theme", user)
		}
		if err := run(t, us, "file/fresh-config", "file/fresh-theme"); err != nil {
			t.Fatal(err)
		}
		if p, err := us["file/fresh-config"].Step.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
			t.Errorf("%s: second plan = %v, %v; want none", user, p.Changes, err)
		}
	}
}

// A link on the way is refused, in root's home too, where WS_USER is root on a cloud VM.
func TestFreshConfigRefusesALink(t *testing.T) {
	for user, home := range map[string]string{"zadykian": "/home/zadykian", "root": "/root"} {
		m, us := load(t, user, home, "amd64")
		m.Links[home+"/.config"] = "/srv/target"
		for _, name := range []string{"file/fresh-config", "file/fresh-theme"} {
			if err := run(t, us, name); err == nil || !strings.Contains(err.Error(), home+"/.config is a link to /srv/target") {
				t.Errorf("%s: %s = %v, want the link refused", user, name, err)
			}
		}
		if len(m.Files) != 0 {
			t.Errorf("%s: %d files, want none", user, len(m.Files))
		}
	}
}
