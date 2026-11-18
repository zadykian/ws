package zed_test

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

const limitsPath = "/etc/security/limits.d/ws-zed.conf"

// loadUnits builds a machine with the account's empty home and its ids, and loads the role's units
// for the values that yaml gives.
func loadUnits(t *testing.T, user, home, yaml string) (*effecttest.Machine, map[string]kind.Unit) {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml", "files/settings.json.tmpl", "files/limits.conf.tmpl"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["zed/"+name] = &fstest.MapFile{Data: text}
	}
	v, err := values.Parse([]byte("WS_USER: " + user + "\n" + yaml))
	if err != nil {
		t.Fatal(err)
	}
	uid := "1000"
	if user == "root" {
		uid = "0"
	}
	m := effecttest.NewMachine()
	m.Results["id -u "+user] = effecttest.Result{Out: []byte(uid + "\n")}
	m.Results["id -g "+user] = effecttest.Result{Out: []byte(uid + "\n")}
	m.Results["getent passwd "+user] = effecttest.Result{Out: []byte(user + ":x:" + uid + ":" + uid + "::" + home + ":/bin/bash\n")}
	m.Dirs[home] = 0o750
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

// The limit names the user, as limits.conf(5) says a `*` line does not apply to root, and takes
// the value of values.yml.
func TestZedNofileNamesTheUserAndTheValue(t *testing.T) {
	m, us := loadUnits(t, "ws", "/home/ws", "WS_ZED_NOFILE: 4096\n")
	if err := runUnits(t, us); err != nil {
		t.Fatal(err)
	}
	want := "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n" +
		"ws soft nofile 4096\n"
	if got := string(m.Files[limitsPath]); got != want {
		t.Errorf("limits file %q, want %q", got, want)
	}
	if owner, set := m.Owners[limitsPath]; set && owner != [2]int{} {
		t.Errorf("owner %v, want root's", owner)
	}
}

func TestZedSettingsOnAnEmptyHome(t *testing.T) {
	m, us := loadUnits(t, "ws", "/home/ws", "")
	if err := runUnits(t, us); err != nil {
		t.Fatal(err)
	}
	if p := "/home/ws/.config/zed/settings.json"; m.Owners[p] != [2]int{1000, 1000} {
		t.Errorf("%s owner %v", p, m.Owners[p])
	}
	for _, dir := range []string{"/home/ws/.config", "/home/ws/.config/zed"} {
		if m.Owners[dir] != [2]int{1000, 1000} {
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

// The servers sit under the user's home, which may be root's on a cloud VM.
func TestZedSettingsNameTheToolPathsUnderTheHome(t *testing.T) {
	for user, home := range map[string]string{"zadykian": "/home/zadykian", "root": "/root"} {
		m, us := loadUnits(t, user, home, "")
		if err := runUnits(t, us); err != nil {
			t.Fatal(err)
		}
		got := string(m.Files[home+"/.config/zed/settings.json"])
		for _, want := range []string{devtools.Gopls(home), devtools.Roslyn(home), devtools.Dlv(home)} {
			if !strings.Contains(got, `"`+want+`"`) {
				t.Errorf("%s: settings lack %q", user, want)
			}
		}
	}
}

// A link on the way is refused, in root's home too, where WS_USER is root on a cloud VM.
func TestZedUnitsRefuseALink(t *testing.T) {
	for user, home := range map[string]string{"ws": "/home/ws", "root": "/root"} {
		m, us := loadUnits(t, user, home, "")
		m.Links[home+"/.config/zed"] = "/srv/target"
		if err := runUnits(t, us); err == nil || !strings.Contains(err.Error(), home+"/.config/zed is a link to /srv/target") {
			t.Errorf("%s: run = %v, want the link at the settings' directory refused", user, err)
		}
		if _, ok := m.Files[home+"/.config/zed/settings.json"]; ok {
			t.Errorf("%s: settings written through the link", user)
		}
	}
}
