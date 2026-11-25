package zed_test

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/roles/devtools"
	"github.com/zadykian/ws/ubuntu/values"
)

const limitsPath = "/etc/security/limits.d/ws-zed.conf"

// loadUnits builds a machine with root's empty home, and loads the role's units for the values that
// yaml gives.
func loadUnits(t *testing.T, yaml string) (*effecttest.Machine, map[string]kind.Unit) {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/settings.json", "files/limits.conf.tmpl"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["zed/"+name] = &fstest.MapFile{Data: text}
	}
	v, err := values.Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0::/root:/bin/bash\n")}
	m.Dirs["/root"] = 0o750
	e := kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v}
	us, err := kind.Load(e, fsys)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]kind.Unit{}
	for _, u := range us {
		byName[u.Name] = u
	}
	if len(byName) != 2 || byName["file/zed-settings"].Name == "" || byName["file/zed-nofile"].Name == "" {
		t.Fatalf("units = %v; want file/zed-nofile and file/zed-settings", kind.Names(us))
	}
	return m, byName
}

func runUnits(t *testing.T, us map[string]kind.Unit) error {
	_, err := step.Run(t.Context(), kind.Steps([]kind.Unit{us["file/zed-nofile"], us["file/zed-settings"]}), &effecttest.Out{}, false)
	return err
}

// The limit names root, as limits.conf(5) says a `*` line does not apply to root, and takes the
// value of values.yml.
func TestZedNofileNamesRootAndTheValue(t *testing.T) {
	m, us := loadUnits(t, "WS_ZED_NOFILE: 4096\n")
	if err := runUnits(t, us); err != nil {
		t.Fatal(err)
	}
	want := "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n" +
		"root soft nofile 4096\n"
	if got := string(m.Files[limitsPath]); got != want {
		t.Errorf("limits file %q, want %q", got, want)
	}
	if owner, set := m.Owners[limitsPath]; set && owner != [2]int{} {
		t.Errorf("owner %v, want root's", owner)
	}
}

func TestZedSettingsOnAnEmptyHome(t *testing.T) {
	m, us := loadUnits(t, "")
	if err := runUnits(t, us); err != nil {
		t.Fatal(err)
	}
	if p := "/root/.config/zed/settings.json"; m.Owners[p] != [2]int{} {
		t.Errorf("%s owner %v", p, m.Owners[p])
	}
	for _, dir := range []string{"/root/.config", "/root/.config/zed"} {
		if m.Owners[dir] != [2]int{} {
			t.Errorf("%s owner %v", dir, m.Owners[dir])
		}
	}
	if err := runUnits(t, us); err != nil {
		t.Fatal(err)
	}
	for name, u := range us {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
			t.Errorf("%s: second plan = %v, %v; want none", name, p.Changes, err)
		}
	}
}

// The servers sit under root's home.
func TestZedSettingsNameTheToolPathsUnderRootsHome(t *testing.T) {
	m, us := loadUnits(t, "")
	if err := runUnits(t, us); err != nil {
		t.Fatal(err)
	}
	got := string(m.Files["/root/.config/zed/settings.json"])
	for _, want := range []string{devtools.GoplsPath, devtools.Roslyn("/root"), devtools.DlvPath} {
		if !strings.Contains(got, `"`+want+`"`) {
			t.Errorf("settings lack %q", want)
		}
	}
}

// A link on the way is refused.
func TestZedUnitsRefuseALink(t *testing.T) {
	m, us := loadUnits(t, "")
	m.Links["/root/.config/zed"] = "/srv/target"
	if err := runUnits(t, us); err == nil || !strings.Contains(err.Error(), "/root/.config/zed is a link to /srv/target") {
		t.Errorf("run = %v, want the link at the settings' directory refused", err)
	}
	if _, ok := m.Files["/root/.config/zed/settings.json"]; ok {
		t.Error("settings written through the link")
	}
}
