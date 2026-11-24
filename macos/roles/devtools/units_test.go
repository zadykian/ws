package devtools_test

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

// userPath is the PATH that a run state's command of the user's runs with.
const userPath = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "

// macUnits loads the role's units on a Mac of the user v.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/paths", "files/gpg-agent.conf"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["devtools/"+name] = &fstest.MapFile{Data: text}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"], m.Dirs["/usr/local"], m.Dirs["/etc/paths.d"] = 0o750, 0o755, 0o755
	m.Owners["/Users/v"] = [2]int{501, 20}
	m.Results[userPath+"mkdir -m 700 /Users/v/.gnupg"] = effecttest.Result{}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, Data: facts.Mac{User: "v", Home: "/Users/v"},
		System: kind.MacOS, User: "v"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

func TestPathNamesHomebrewsAndGosBin(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/Users/v/.gnupg"], m.Owners["/Users/v/.gnupg"] = 0o700, [2]int{501, 20}
	units := macUnits(t, m)
	if _, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/etc/paths.d/ws"]); got != "/opt/homebrew/bin\n/usr/local/go/bin\n" || m.Owners["/etc/paths.d/ws"] != [2]int{} {
		t.Errorf("/etc/paths.d/ws = %q, owner %v; want both bins, root's", got, m.Owners["/etc/paths.d/ws"])
	}
}

func TestGoIsLinkedToMisesInstall(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/Users/v/.gnupg"], m.Owners["/Users/v/.gnupg"] = 0o700, [2]int{501, 20}
	units := macUnits(t, m)
	if _, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile("mise.macos.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Tools map[string]string `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	want := "/Users/v/.local/share/mise/installs/go/" + c.Tools["go"]
	if got := m.Links["/usr/local/go"]; got != want {
		t.Errorf("/usr/local/go -> %q, want %q", got, want)
	}
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

func TestGpgAgentSeedsPinentryMac(t *testing.T) {
	m := effecttest.NewMachine()
	units := macUnits(t, m)
	conf := "/Users/v/.gnupg/gpg-agent.conf"
	for _, u := range units {
		if u.Name != "state/gnupg-dir" && u.Name != "file/gpg-agent-conf" {
			continue
		}
		p, err := u.Step.Plan(t.Context())
		if err == nil {
			err = u.Step.Apply(t.Context(), p)
		}
		if err != nil {
			t.Fatalf("%s: %v", u.Name, err)
		}
	}
	if got := string(m.Files[conf]); !strings.Contains(got, "\npinentry-program /opt/homebrew/bin/pinentry-mac\n") ||
		m.Modes[conf] != 0o600 || m.Owners[conf] != [2]int{501, 20} {
		t.Errorf("%s = %q, mode %o, owner %v; want pinentry-mac, 0600, v's", conf, got, m.Modes[conf], m.Owners[conf])
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasSuffix(c, "mkdir -m 700 /Users/v/.gnupg") }) {
		t.Errorf("calls %q lack mkdir -m 700 of ~/.gnupg", m.Calls)
	}
}

// A seed is the user's after its write: a changed file stays, and the plan changes nothing.
func TestGpgAgentKeepsTheUsersFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/Users/v/.gnupg/gpg-agent.conf"] = []byte("default-cache-ttl 60\n")
	m.Dirs["/Users/v/.gnupg"], m.Owners["/Users/v/.gnupg"] = 0o700, [2]int{501, 20}
	for _, u := range macUnits(t, m) {
		if u.Name != "state/gnupg-dir" && u.Name != "file/gpg-agent-conf" {
			continue
		}
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}
