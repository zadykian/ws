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

// machine is a machine with the account zadykian, 1000:1000, and its env for the values of text.
func machine(t *testing.T, text string) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["id -u zadykian"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g zadykian"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -u root"] = effecttest.Result{Out: []byte("0\n")}
	m.Results["id -g root"] = effecttest.Result{Out: []byte("0\n")}
	m.Results["getent passwd zadykian"] = effecttest.Result{Out: []byte("zadykian:x:1000:1000::/home/zadykian:/bin/bash\n")}
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/home/zadykian"], m.Owners["/home/zadykian"] = 0o750, [2]int{1000, 1000}
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

// The same units for an account and for root, and each command runs as WS_USER in its home, though
// systemd is not running, as in CI's container. roslyn's finds dotnet's tools first on its PATH.
func TestDevtoolsUnitsForAnAccountAndRoot(t *testing.T) {
	var names []string
	for _, a := range []struct{ values, as, home string }{
		{"", "runuser -u zadykian -- env -i -C /home/zadykian HOME=/home/zadykian PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS ", "/home/zadykian"},
		{"WS_USER: root\n", "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS ", "/root"},
	} {
		m, e := machine(t, a.values)
		us := units(t, e)
		if names == nil {
			names = kind.Names(us)
		} else if got := kind.Names(us); !slices.Equal(got, names) {
			t.Errorf("units for root = %q, want the account's %q", got, names)
		}
		for _, chain := range [][]string{
			{"file/devtools-npm-lock", "file/devtools-npm-package", "state/devtools-npm", "file/devtools-sql-formatter"},
			{"state/devtools-gh-telemetry", "state/devtools-gh-token-group", "state/devtools-gh-login"},
		} {
			for i := 1; i < len(chain); i++ {
				if slices.Index(names, chain[i-1]) > slices.Index(names, chain[i]) {
					t.Errorf("order %q: %s runs before %s", names, chain[i], chain[i-1])
				}
			}
		}
		tools := "PATH=" + a.home + "/.dotnet/tools:" + strings.TrimPrefix(systemPath, "PATH=")
		for name, path := range map[string]string{"state/devtools-npm": systemPath, "state/devtools-gh-telemetry": systemPath,
			"state/devtools-roslyn": tools} {
			m.Calls = nil
			if got := changes(t, units(t, e, name)[0]); len(got) != 1 {
				t.Errorf("%s: plan = %q, want the command, as its check fails", name, got)
			}
			want := a.as + path + " sh -c"
			if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, want) }) {
				t.Errorf("%s: calls %q, want the check as %q", name, m.Calls, want)
			}
		}
		// gh's login is ws's, as root, which runs gh as the account.
		m.Calls = nil
		changes(t, units(t, e, "state/devtools-gh-login")[0])
		want := "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " + systemPath +
			" /usr/local/bin/ws server ubuntu devtools gh-login --check"
		if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasSuffix(c, want) }) {
			t.Errorf("gh-login: calls %q, want the check as %q", m.Calls, want)
		}
	}
}

// The account's files go in as its own, with the directories that a new home lacks.
func TestNpmFilesAreTheAccountsWithTheirDirectories(t *testing.T) {
	m, e := machine(t, "")
	write(t, e, units(t, e, "file/devtools-npm-lock", "file/devtools-npm-package"))
	dir := "/home/zadykian/.local/lib/language-servers"
	for _, p := range []string{"/home/zadykian/.local", "/home/zadykian/.local/lib", dir} {
		if m.Dirs[p] != 0o755 || m.Owners[p] != [2]int{1000, 1000} {
			t.Errorf("%s: mode %o, owner %v; want 755, the account's", p, m.Dirs[p], m.Owners[p])
		}
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		want, err := fs.ReadFile(os.DirFS("files/npm"), name)
		if err != nil {
			t.Fatal(err)
		}
		if p := dir + "/" + name; string(m.Files[p]) != string(want) || m.Owners[p] != [2]int{1000, 1000} {
			t.Errorf("%s: owner %v, or not files/npm's text", p, m.Owners[p])
		}
	}
}

func TestRustupCreatesItsBinaryOnce(t *testing.T) {
	m, e := machine(t, "")
	u := units(t, e, "state/devtools-rustup")[0]
	if got := changes(t, u); len(got) != 1 || !strings.Contains(got[0], "sh.rustup.rs") {
		t.Errorf("plan = %q, want the installer", got)
	}
	m.Files["/home/zadykian/.cargo/bin/rustup"] = nil
	if got := changes(t, u); got != nil {
		t.Errorf("plan with rustup there = %q, want none", got)
	}
}
