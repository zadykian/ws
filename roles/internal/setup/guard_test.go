package setup_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/internal/setup"
)

// guarded returns a machine whose directories above and tree are root's alone, with a file in the
// tree and one in a directory of it.
func guarded() (*effecttest.Machine, setup.Guard) {
	m := effecttest.NewMachine()
	for _, d := range []string{"/", "/usr", "/opt", "/opt/ws", "/opt/ws/roles"} {
		m.Dirs[d] = 0o755
	}
	m.Files["/opt/ws/mise.toml"] = nil
	m.Files["/opt/ws/roles/mise.toml"] = nil
	return m, setup.Guard{M: m, Dir: "/opt/ws", Above: []string{"/", "/usr", "/opt"}}
}

func TestGuardPassesATreeThatRootOwnsAlone(t *testing.T) {
	_, g := guarded()
	changes, err := g.Plan(t.Context())
	if err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v", changes, err)
	}
}

func TestGuardPassesATreeThatDoesNotExist(t *testing.T) {
	m, g := guarded()
	delete(m.Dirs, "/opt/ws")
	if _, err := g.Plan(t.Context()); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestGuardRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(m *effecttest.Machine)
		want   string
	}{
		"a tree that is a link": {func(m *effecttest.Machine) {
			delete(m.Dirs, "/opt/ws")
			m.Links["/opt/ws"] = "/home/other/ws"
		}, "/opt/ws is a link"},
		"a file of another owner": {func(m *effecttest.Machine) {
			m.Owners["/opt/ws/roles/mise.toml"] = [2]int{1000, 1000}
		}, "/opt/ws/roles/mise.toml is of uid 1000"},
		"a file that a group writes": {func(m *effecttest.Machine) {
			m.Modes["/opt/ws/mise.toml"] = 0o664
		}, "/opt/ws/mise.toml is of uid 0, with mode 664"},
		"a directory that others write": {func(m *effecttest.Machine) {
			m.Dirs["/opt/ws/roles"] = 0o757
		}, "/opt/ws/roles is of uid 0, with mode 757"},
		"the tree's own mode": {func(m *effecttest.Machine) {
			m.Dirs["/opt/ws"] = 0o775
		}, "/opt/ws is of uid 0, with mode 775"},
		"a directory above that others write": {func(m *effecttest.Machine) {
			m.Dirs["/opt"] = 0o777
		}, "/opt is not root's alone, and mise ignores the plugin"},
		"a directory above of another owner": {func(m *effecttest.Machine) {
			m.Owners["/usr"] = [2]int{1000, 1000}
		}, "/usr is not root's alone"},
	} {
		t.Run(name, func(t *testing.T) {
			m, g := guarded()
			tc.change(m)
			_, err := g.Plan(t.Context())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestGuardPassesALinkToAFileOfRoots(t *testing.T) {
	m, g := guarded()
	m.Links["/opt/ws/roles/latest"] = "/opt/ws/mise.toml"
	if _, err := g.Plan(t.Context()); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestGuardRefusesALinkByItsTarget(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(m *effecttest.Machine)
		want   string
	}{
		"a target of another owner": {func(m *effecttest.Machine) {
			m.Files["/home/other/config"] = nil
			m.Owners["/home/other/config"] = [2]int{1000, 1000}
			m.Links["/opt/ws/.git/config"] = "/home/other/config"
		}, "/opt/ws/.git/config links to /home/other/config, which is of uid 1000"},
		"a target that a group writes": {func(m *effecttest.Machine) {
			m.Files["/srv/hook"] = nil
			m.Modes["/srv/hook"] = 0o664
			m.Links["/opt/ws/roles/hook"] = "/srv/hook"
		}, "which is of uid 0, with mode 664"},
		"a target that is missing": {func(m *effecttest.Machine) {
			m.Links["/opt/ws/roles/hook"] = "/home/other/hook"
		}, "/opt/ws/roles/hook links to /home/other/hook, which is missing"},
	} {
		t.Run(name, func(t *testing.T) {
			m, g := guarded()
			m.Dirs["/opt/ws/.git"] = 0o755
			tc.change(m)
			_, err := g.Plan(t.Context())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestGuardRefusesADirectoryAboveThatIsALink(t *testing.T) {
	m, g := guarded()
	m.Dirs["/srv/opt"] = 0o755
	m.Links["/opt"] = "/srv/opt"
	if _, err := g.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "/opt is a link") {
		t.Errorf("err = %v", err)
	}
}

func TestGuardWantsTheFileOfATreeThatRunsAsItStands(t *testing.T) {
	m, g := guarded()
	g.Holds = "mise.toml"
	if _, err := g.Plan(t.Context()); err != nil {
		t.Errorf("a tree with mise.toml: %v", err)
	}
	delete(m.Files, "/opt/ws/mise.toml")
	if _, err := g.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "/opt/ws holds no mise.toml") {
		t.Errorf("a tree without it: %v", err)
	}
	delete(m.Dirs, "/opt/ws")
	if _, err := g.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "holds no mise.toml") {
		t.Errorf("no tree: %v", err)
	}
}
