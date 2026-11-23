package fresh_test

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools"
	"github.com/zadykian/ws/roles/server/values"
)

// loadUnits builds a machine with root's home and the architecture, and loads the role's units.
func loadUnits(t *testing.T, arch string) (*effecttest.Machine, map[string]kind.Unit) {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/config.json", "files/themes/jetbrains-dark.json"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["fresh/"+name] = &fstest.MapFile{Data: text}
	}
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte(arch + "\n")}
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0::/root:/bin/bash\n")}
	m.Dirs["/root"] = 0o750
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

func runUnits(t *testing.T, us map[string]kind.Unit, names ...string) error {
	var list []kind.Unit
	for _, n := range names {
		list = append(list, us[n])
	}
	_, err := step.Run(t.Context(), kind.Steps(list), &effecttest.Out{}, false)
	return err
}

// The .deb is the pin: the release's file for the architecture, checked by its sum.
func TestFreshDebIsThePin(t *testing.T) {
	_, us := loadUnits(t, "arm64")
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
	_, us := loadUnits(t, "riscv64")
	if _, err := us["deb/fresh"].Step.Plan(t.Context()); err == nil {
		t.Error("no error")
	}
}

// The servers sit under root's home.
func TestFreshConfigNamesRootsServers(t *testing.T) {
	m, us := loadUnits(t, "amd64")
	if err := runUnits(t, us, "file/fresh-config", "file/fresh-theme"); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/root/.config/fresh/config.json"]); !strings.Contains(got, `"`+devtools.Roslyn("/root")+`"`) {
		t.Error("config.json lacks root's roslyn-language-server")
	}
	if _, ok := m.Files["/root/.config/fresh/themes/jetbrains-dark.json"]; !ok {
		t.Error("no theme")
	}
	if err := runUnits(t, us, "file/fresh-config", "file/fresh-theme"); err != nil {
		t.Fatal(err)
	}
	if p, err := us["file/fresh-config"].Step.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("second plan = %v, %v; want none", p.Changes, err)
	}
}

// A link on the way is refused.
func TestFreshConfigRefusesALink(t *testing.T) {
	m, us := loadUnits(t, "amd64")
	m.Links["/root/.config"] = "/srv/target"
	for _, name := range []string{"file/fresh-config", "file/fresh-theme"} {
		if err := runUnits(t, us, name); err == nil || !strings.Contains(err.Error(), "/root/.config is a link to /srv/target") {
			t.Errorf("%s = %v, want the link refused", name, err)
		}
	}
	if len(m.Files) != 0 {
		t.Errorf("%d files, want none", len(m.Files))
	}
}
