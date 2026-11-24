package setup_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/internal/setup"
)

// settle applies what each resource plans, as a run does, and returns the changes' count.
func settle(t *testing.T, rs []resource.Resource) int {
	n := 0
	for _, r := range rs {
		changes, err := r.Plan(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range changes {
			if err := c.Apply(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
		n += len(changes)
	}
	return n
}

func TestPluginLinksTheTreesPluginAndSettlesAtOnce(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/usr/share"] = 0o755
	rs := setup.Plugin(m, setup.TreeDir, setup.PluginDir, 0)
	if n := settle(t, rs); n != 3 {
		t.Errorf("first run made %d changes, want the two directories and the link", n)
	}
	if got := m.Links["/usr/share/mise/plugins/ws"]; got != "/opt/ws/plugins/ws" {
		t.Errorf("link = %q", got)
	}
	for _, dir := range []string{"/usr/share/mise", "/usr/share/mise/plugins"} {
		if m.Dirs[dir] != 0o755 {
			t.Errorf("%s has mode %o", dir, m.Dirs[dir])
		}
	}
	if n := settle(t, rs); n != 0 {
		t.Errorf("second run made %d changes", n)
	}
}

func TestPluginMendsAWrongLinkAndAWideDirectory(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/usr/share"] = 0o755
	m.Dirs["/usr/share/mise"] = 0o775
	m.Dirs["/usr/share/mise/plugins"] = 0o755
	m.Links["/usr/share/mise/plugins/ws"] = "/home/other/plugins/ws"
	if n := settle(t, setup.Plugin(m, setup.TreeDir, setup.PluginDir, 0)); n != 2 {
		t.Errorf("made %d changes, want the mode and the link", n)
	}
	if got := m.Links["/usr/share/mise/plugins/ws"]; got != "/opt/ws/plugins/ws" {
		t.Errorf("link = %q", got)
	}
}
