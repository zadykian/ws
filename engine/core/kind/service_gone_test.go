package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// goneService is a service whose two files go: its unit file and a script.
const goneService = "[service.a]\nunit = \"a.service\"\nrun = \"stopped\"\nin_force = [\"show\"]\nloads = [\"load\"]\n" +
	"[[service.a.file]]\npath = \"/usr/local/lib/a/a.sh\"\nabsent = true\n" +
	"[[service.a.file]]\npath = \"/etc/systemd/system/a.service\"\nabsent = true\n"

// goneEnv is a machine where an earlier run left the service and its files.
func goneEnv() (Env, *effecttest.Machine, *effecttest.Units, *effecttest.Out) {
	e, m, u, out := testEnv()
	m.Dirs["/usr/local/lib/a"], m.Dirs["/etc/systemd/system"] = 0o755, 0o755
	m.Files["/usr/local/lib/a/a.sh"], m.Files["/etc/systemd/system/a.service"] = []byte("x"), []byte("[Unit]\n")
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}}
	return e, m, u, out
}

func TestServiceGoneStopsAndDisablesBeforeItsFiles(t *testing.T) {
	e, m, u, _ := goneEnv()
	units := loadOne(t, e, goneService, nil)
	want := []string{"disable a.service", "stop a.service", "remove /usr/local/lib/a/a.sh",
		"remove /etc/systemd/system/a.service", "reload systemd's unit files"}
	if got := headlines(t, units[0].Step); !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/etc/systemd/system/a.service"]; ok || !slices.Equal(u.Calls, []string{"disable a.service", "stop a.service", "daemon-reload"}) {
		t.Errorf("unit file there = %v, calls %q; want it gone after the disable and the stop", ok, u.Calls)
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("a second plan = %q, want none", got)
	}
}

func TestServiceGoneRefusesEnabledOrRunning(t *testing.T) {
	e, _, _, _ := testEnv()
	const part = "[[service.a.file]]\npath = \"/etc/a.conf\"\nabsent = true\n"
	for _, fields := range []string{"enabled = true\n", "run = \"running\"\n", "run = \"triggered\"\n"} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[service.a]\nunit = \"a.service\"\n" + fields + part}))
		if err == nil || !strings.Contains(err.Error(), "whose files are all absent") {
			t.Errorf("load of\n%s= %v, want a refusal", fields, err)
		}
	}
}

func TestServiceGonePlansNoDrift(t *testing.T) {
	e, m, _, _ := goneEnv()
	units := loadOne(t, e, goneService, nil)
	headlines(t, units[0].Step)
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasSuffix(c, " show") || strings.HasSuffix(c, " load") }) {
		t.Errorf("calls = %q, want no comparison", m.Calls)
	}
}
