package devtools_test

import (
	"context"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/secrets"
	"github.com/zadykian/ws/roles/server/values"
)

const (
	systemPath = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	ghFile     = "/etc/ws/gh-token"
)

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

// machine is a machine with the account zadykian, 1000:1000, and its env for the values of text,
// whose secrets come from load.
func machine(t *testing.T, text string, load func(context.Context) (any, error)) (*effecttest.Machine, kind.Env) {
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
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Secrets: load}
}

// token returns secrets whose token.github is tok.
func token(tok string) func(context.Context) (any, error) {
	return func(context.Context) (any, error) {
		return secrets.Secrets{Token: secrets.Tokens{GitHub: tok}, Whole: true}, nil
	}
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

// telemetryOff makes the telemetry's check pass on m, so that the token's file, behind it, applies.
func telemetryOff(t *testing.T, m *effecttest.Machine, e kind.Env) {
	m.Calls = nil
	changes(t, units(t, e, "state/devtools-gh-telemetry")[0])
	for _, c := range m.Calls {
		if _, line, ok := strings.Cut(c, " "); ok && strings.Contains(line, "gh config get telemetry") {
			m.Results[line] = effecttest.Result{}
		}
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
		{"", "runuser -u zadykian -- env -C /home/zadykian HOME=/home/zadykian env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS ", "/home/zadykian"},
		{"WS_USER: root\n", "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS ", "/root"},
	} {
		m, e := machine(t, a.values, nil)
		us := units(t, e)
		if names == nil {
			names = kind.Names(us)
		} else if got := kind.Names(us); !slices.Equal(got, names) {
			t.Errorf("units for root = %q, want the account's %q", got, names)
		}
		for _, chain := range [][]string{
			{"file/devtools-npm-lock", "file/devtools-npm-package", "state/devtools-npm", "file/devtools-sql-formatter"},
			{"state/devtools-gh-telemetry", "file/devtools-gh-token", "state/devtools-gh-login"},
		} {
			for i := 1; i < len(chain); i++ {
				if slices.Index(names, chain[i-1]) > slices.Index(names, chain[i]) {
					t.Errorf("order %q: %s runs before %s", names, chain[i], chain[i-1])
				}
			}
		}
		tools := "PATH=" + a.home + "/.dotnet/tools:" + strings.TrimPrefix(systemPath, "PATH=")
		for name, path := range map[string]string{"state/devtools-npm": systemPath, "state/devtools-gh-telemetry": systemPath,
			"state/devtools-gh-login": systemPath, "state/devtools-roslyn": tools} {
			m.Calls = nil
			if got := changes(t, units(t, e, name)[0]); len(got) != 1 {
				t.Errorf("%s: plan = %q, want the command, as its check fails", name, got)
			}
			want := a.as + path + " sh -c"
			if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, want) }) {
				t.Errorf("%s: calls %q, want the check as %q", name, m.Calls, want)
			}
		}
	}
}

// The account's files go in as its own, with the directories that a new home lacks.
func TestNpmFilesAreTheAccountsWithTheirDirectories(t *testing.T) {
	m, e := machine(t, "", nil)
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
	m, e := machine(t, "", nil)
	u := units(t, e, "state/devtools-rustup")[0]
	if got := changes(t, u); len(got) != 1 || !strings.Contains(got[0], "sh.rustup.rs") {
		t.Errorf("plan = %q, want the installer", got)
	}
	m.Files["/home/zadykian/.cargo/bin/rustup"] = nil
	if got := changes(t, u); got != nil {
		t.Errorf("plan with rustup there = %q, want none", got)
	}
}

// The token file is root's, read by the account's group, and holds ws's mark and the token on its
// last line.
func TestGHTokenFileEndsWithTheToken(t *testing.T) {
	m, e := machine(t, "", token("dummy-token"))
	telemetryOff(t, m, e)
	us := units(t, e, "state/devtools-gh-telemetry", "file/devtools-gh-token")
	write(t, e, us)
	lines := strings.Split(strings.TrimSuffix(string(m.Files[ghFile]), "\n"), "\n")
	if !file.Ours(m.Files[ghFile]) || lines[len(lines)-1] != "dummy-token" {
		t.Errorf("%s = %q, want the mark first and the token last", ghFile, lines)
	}
	if m.Owners[ghFile] != [2]int{0, 1000} || m.Modes[ghFile] != 0o640 {
		t.Errorf("owner %v, mode %o; want 0:1000, 640", m.Owners[ghFile], m.Modes[ghFile])
	}
	if got := changes(t, us[1]); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "dummy-token") {
			t.Errorf("call %q holds the token", c)
		}
	}
	m, e = machine(t, "", token(""))
	m.Files[ghFile] = []byte(file.Header("#") + "\nold-token\n")
	telemetryOff(t, m, e)
	write(t, e, units(t, e, "state/devtools-gh-telemetry", "file/devtools-gh-token"))
	if _, ok := m.Files[ghFile]; ok {
		t.Error("no token.github left the file")
	}
}

// mise writes only the units whose read is not in sync. A new token's file must take the login
// along on the run that writes it, so the login's read names the file while the file has changes.
func TestANewTokenTakesTheLoginAlong(t *testing.T) {
	m, e := machine(t, "", token("dummy-token"))
	telemetryOff(t, m, e)
	reports, err := step.Read(t.Context(), kind.Steps(units(t, e, "state/devtools-gh-telemetry",
		"file/devtools-gh-token", "state/devtools-gh-login")))
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(reports, func(r step.Report) bool { return r.Step == "state/devtools-gh-login" })
	if i < 0 || reports[i].After != "file/devtools-gh-token" {
		t.Errorf("reports %+v: want the login read after the token's file", reports)
	}
}

func TestGHTokenWithoutSecretsPlansNothing(t *testing.T) {
	m, e := machine(t, "", func(context.Context) (any, error) { return nil, kind.ErrNoSecrets })
	m.Files[ghFile] = []byte("by hand\n")
	if got := changes(t, units(t, e, "file/devtools-gh-token")[0]); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}
