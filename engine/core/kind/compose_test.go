package kind

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	composeSpecText = "[compose.p]\ndir = \"/srv/p\"\nproject = \"p\"\n"
	composePs       = "docker compose --project-name p ps --status running --services"
	composeUpLine   = "docker compose --project-directory /srv/p --project-name p up --detach --quiet-pull --remove-orphans"
	composeRecreate = composeUpLine + " --force-recreate"
	composeSumFile  = "/var/lib/ws/compose/p"
	recreateP       = "recreate the containers of p"
	startP          = "start the containers of p that are down"
)

// composeMachine is a machine whose project p, in /srv/p, has a service s that keeps running
// and a one-shot once. Its compose lists running as the services that run.
func composeMachine(t *testing.T, running ...string) (*effecttest.Machine, []Unit) {
	e, m, _, _ := testEnv()
	m.Dirs["/srv/p"] = 0o755
	m.Dirs["/srv/p/conf"] = 0o755
	m.Dirs["/srv/p/empty"] = 0o755
	m.Files["/srv/p/compose.yaml"] = []byte("services:\n  s:\n    image: busybox\n    restart: unless-stopped\n" +
		"  once:\n    image: busybox\n    restart: on-failure\n")
	m.Files["/srv/p/conf/a.yaml"] = []byte("a: 1\n")
	m.Results[composePs] = effecttest.Result{Out: []byte(strings.Join(running, "\n") + "\n")}
	m.Results[composeUpLine] = effecttest.Result{}
	m.Results[composeRecreate] = effecttest.Result{}
	return m, loadOne(t, e, composeSpecText, nil)
}

// upped is composeMachine after a good up, with its sum kept.
func upped(t *testing.T, running ...string) (*effecttest.Machine, []Unit) {
	m, units := composeMachine(t, running...)
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	m.Calls = nil
	return m, units
}

func TestComposeUpsWithoutASum(t *testing.T) {
	m, units := composeMachine(t, "s")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{recreateP}) {
		t.Fatalf("plan = %q, want the recreation", got)
	}
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "root "+composeRecreate) {
		t.Errorf("calls = %q, want the up that recreates", m.Calls)
	}
	sum, err := projectSum(m, "/srv/p")
	if err != nil {
		t.Fatal(err)
	}
	if got, a := string(m.Files[composeSumFile]), m.Root[composeSumFile]; got != sum+"\n" || a != (effect.Attrs{Mode: 0o600}) {
		t.Errorf("kept sum = %q, %+v; want %q, root's at 600", got, a, sum+"\n")
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestComposeKeepsItsSum(t *testing.T) {
	m, units := upped(t, "s")
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
	if !slices.Contains(m.Calls, "root "+composePs) {
		t.Errorf("calls = %q, want compose asked for the running services", m.Calls)
	}
}

func TestComposeRecreatesWhenAFileChanges(t *testing.T) {
	for name, change := range map[string]func(m *effecttest.Machine){
		"a byte of a nested file": func(m *effecttest.Machine) { m.Files["/srv/p/conf/a.yaml"] = []byte("a: 2\n") },
		"a new file":              func(m *effecttest.Machine) { m.Files["/srv/p/empty/b"] = nil },
		"a removed file":          func(m *effecttest.Machine) { delete(m.Files, "/srv/p/conf/a.yaml") },
		"a new link":              func(m *effecttest.Machine) { m.Links["/srv/p/l"] = "/srv/p/compose.yaml" },
		"a moved file": func(m *effecttest.Machine) {
			m.Files["/srv/p/conf/b.yaml"] = m.Files["/srv/p/conf/a.yaml"]
			delete(m.Files, "/srv/p/conf/a.yaml")
		},
	} {
		m, units := upped(t, "s")
		change(m)
		if got := headlines(t, units[0].Step); !slices.Equal(got, []string{recreateP}) {
			t.Errorf("plan after %s = %q, want the recreation", name, got)
		}
	}
}

func TestComposeRecreatesForARootOnlyFile(t *testing.T) {
	m, units := upped(t, "s")
	m.Files["/srv/p/root.env"] = []byte("PASSWORD=x\n")
	m.Root["/srv/p/root.env"] = effect.Attrs{Mode: 0o600}
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{recreateP}) {
		t.Errorf("plan = %q, want the recreation", got)
	}
	if !slices.Contains(m.Calls, "read root /srv/p/root.env") {
		t.Errorf("calls = %q, want root.env read as root", m.Calls)
	}
}

func TestComposeUpsWhenAServiceIsDown(t *testing.T) {
	m, units := upped(t)
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{startP}) {
		t.Fatalf("plan = %q, want the up of what is down", got)
	}
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "root "+composeUpLine) || slices.Contains(m.Calls, "root "+composeRecreate) {
		t.Errorf("calls = %q, want an up without the recreation", m.Calls)
	}
}

func TestComposeIgnoresOneShots(t *testing.T) {
	_, units := upped(t, "s")
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan with the one-shot once down = %q, want none", got)
	}
}

func TestComposeKeepsTheOldSumOnAFailedUp(t *testing.T) {
	m, units := upped(t, "s")
	old := string(m.Files[composeSumFile])
	m.Files["/srv/p/conf/a.yaml"] = []byte("a: 2\n")
	m.Results[composeRecreate] = effecttest.Result{Err: errors.New("pull failed")}
	if err := applyOnce(t, units[0].Step); err == nil {
		t.Fatal("apply succeeded, want the up's error")
	}
	if got := string(m.Files[composeSumFile]); got != old {
		t.Errorf("kept sum = %q, want the old %q", got, old)
	}
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{recreateP}) {
		t.Errorf("plan after the failed up = %q, want the recreation again", got)
	}
}

func TestComposeRefusesABadSpec(t *testing.T) {
	e, _, _, _ := testEnv()
	for fields, want := range map[string]string{
		"dir = \"srv/p\"\nproject = \"p\"\n":      `dir "srv/p" is no clean absolute path`,
		"dir = \"/srv/../p\"\nproject = \"p\"\n":  `dir "/srv/../p" is no clean absolute path`,
		"dir = \"/\"\nproject = \"p\"\n":          `dir "/" is no clean absolute path`,
		"project = \"p\"\n":                       `dir "" is no clean absolute path`,
		"dir = \"/srv/p\"\nproject = \"a/b\"\n":   `project "a/b" is no compose project's name`,
		"dir = \"/srv/p\"\nproject = \"P\"\n":     `project "P" is no compose project's name`,
		"dir = \"/srv/p\"\nproject = \"-p\"\n":    `project "-p" is no compose project's name`,
		"dir = \"/srv/p\"\n":                      `project "" is no compose project's name`,
		"dir = \"/srv/p\"\nproject = \"p q\"\n":   `project "p q" is no compose project's name`,
		"dir = \"/srv/p/\"\nproject = \"p\"\n":    `dir "/srv/p/" is no clean absolute path`,
		"dir = \"/srv/p\"\nproject = \"p_1-a\"\n": "",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[compose.a]\n" + fields}))
		if want == "" && err != nil || want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
			t.Errorf("load of\n%s= %v, want %q", fields, err, want)
		}
	}
}

func TestComposeArgs(t *testing.T) {
	p := composeProject{dir: "/srv/p", project: "p"}
	for _, c := range []struct {
		up   composeUp
		want string
	}{{composeUp{p, false}, composeUpLine}, {composeUp{p, true}, composeRecreate}} {
		if got := "docker " + strings.Join(c.up.args(), " "); got != c.want {
			t.Errorf("args = %q, want %q", got, c.want)
		}
	}
}
