package devtools_test

import (
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/values"
)

const systemPath = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// role is devtools' spec with its files, as kind.Load finds a role's.
func role(t *testing.T) fstest.MapFS {
	fsys, dir := fstest.MapFS{}, os.DirFS(".")
	err := fs.WalkDir(dir, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || (name != "units.toml.tmpl" && !strings.HasPrefix(name, "files/")) {
			return err
		}
		text, err := fs.ReadFile(dir, name)
		fsys["devtools/"+name] = &fstest.MapFile{Data: text}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

// machine is a machine with root's home, and its env for the values of an empty file.
func machine(t *testing.T) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/root"], m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o700, 0o755, 0o755
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Self: "/usr/local/bin/ws"}
}

// units loads devtools' units on e, and keeps those named, or all where none is.
func units(t *testing.T, e kind.Env, names ...string) []kind.Unit {
	all, err := kind.Load(e, role(t))
	if err != nil {
		t.Fatal(err)
	}
	return slices.DeleteFunc(all, func(u kind.Unit) bool { return len(names) > 0 && !slices.Contains(names, u.Name) })
}

// write runs the units as ws unit write does.
func write(t *testing.T, e kind.Env, us []kind.Unit) {
	if _, err := step.Run(t.Context(), kind.Steps(us), e.Out, false); err != nil {
		t.Fatal(err)
	}
}

// changes returns the headlines of the unit's plan.
func changes(t *testing.T, u kind.Unit) []string {
	p, err := u.Step.Plan(t.Context())
	if err != nil {
		t.Fatalf("plan %s: %v", u.Name, err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, step.Headline(c))
	}
	return got
}

// Each command runs as root in its home, though systemd is not running, as in CI's container.
// roslyn's finds dotnet's tools first on its PATH.
func TestDevtoolsUnitsRunAsRoot(t *testing.T) {
	m, e := machine(t)
	names := kind.Names(units(t, e))
	for _, chain := range [][]string{
		{"file/devtools-npm-lock", "file/devtools-npm-package", "state/devtools-npm", "file/devtools-sql-formatter"},
		{"state/devtools-gh-telemetry", "state/devtools-gh-login"},
	} {
		for i := 1; i < len(chain); i++ {
			if slices.Index(names, chain[i-1]) > slices.Index(names, chain[i]) {
				t.Errorf("order %q: %s runs before %s", names, chain[i], chain[i-1])
			}
		}
	}
	const as = "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS "
	tools := "PATH=/root/.dotnet/tools:" + strings.TrimPrefix(systemPath, "PATH=")
	for name, path := range map[string]string{"state/devtools-npm": systemPath, "state/devtools-gh-telemetry": systemPath,
		"state/devtools-roslyn": tools} {
		m.Calls = nil
		if got := changes(t, units(t, e, name)[0]); len(got) != 1 {
			t.Errorf("%s: plan = %q, want the command, as its check fails", name, got)
		}
		want := as + path + " sh -c"
		if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, want) }) {
			t.Errorf("%s: calls %q, want the check as %q", name, m.Calls, want)
		}
	}
	// gh's login is ws's own, which reads the token and runs gh.
	m.Calls = nil
	changes(t, units(t, e, "state/devtools-gh-login")[0])
	want := as + systemPath + " /usr/local/bin/ws server devtools gh-login --check"
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasSuffix(c, want) }) {
		t.Errorf("gh-login: calls %q, want the check as %q", m.Calls, want)
	}
}

// The npm files go in as root's, with the directories that a new home lacks.
func TestNpmFilesAreRootsWithTheirDirectories(t *testing.T) {
	m, e := machine(t)
	write(t, e, units(t, e, "file/devtools-npm-lock", "file/devtools-npm-package"))
	dir := "/root/.local/lib/language-servers"
	for _, p := range []string{"/root/.local", "/root/.local/lib", dir} {
		if m.Dirs[p] != 0o755 || m.Owners[p] != [2]int{} {
			t.Errorf("%s: mode %o, owner %v; want 755, root's", p, m.Dirs[p], m.Owners[p])
		}
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		want, err := fs.ReadFile(os.DirFS("files/npm"), name)
		if err != nil {
			t.Fatal(err)
		}
		if p := dir + "/" + name; string(m.Files[p]) != string(want) || m.Owners[p] != [2]int{} {
			t.Errorf("%s: owner %v, or not files/npm's text", p, m.Owners[p])
		}
	}
}

func TestRustupCreatesItsBinaryOnce(t *testing.T) {
	m, e := machine(t)
	u := units(t, e, "state/devtools-rustup")[0]
	if got := changes(t, u); len(got) != 1 || !strings.Contains(got[0], "sh.rustup.rs") {
		t.Errorf("plan = %q, want the installer", got)
	}
	m.Files["/root/.cargo/bin/rustup"] = nil
	if got := changes(t, u); got != nil {
		t.Errorf("plan with rustup there = %q, want none", got)
	}
}
