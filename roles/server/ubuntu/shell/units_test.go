package shell_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/values"
)

const envPath = "/etc/ws/env"

// role is the shell's spec, as kind.Load finds a role's.
func role(t *testing.T) fstest.MapFS {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return fstest.MapFS{"shell/units.toml.tmpl": &fstest.MapFile{Data: text}}
}

// machine is a machine with the account zadykian, 1000:1000, and /etc/ws, and its env for the
// default values.
func machine(t *testing.T) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["id -u zadykian"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g zadykian"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["getent passwd zadykian"] = effecttest.Result{Out: []byte("zadykian:x:1000:1000::/home/zadykian:/bin/bash\n")}
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/home/zadykian"], m.Owners["/home/zadykian"] = 0o750, [2]int{1000, 1000}
	m.Dirs["/root"], m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o700, 0o755, 0o755
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v}
}

// units loads the shell's units on e, and keeps those named.
func units(t *testing.T, e kind.Env, names ...string) []kind.Unit {
	all, err := kind.Load(e, role(t))
	if err != nil {
		t.Fatal(err)
	}
	return slices.DeleteFunc(all, func(u kind.Unit) bool { return len(names) > 0 && !slices.Contains(names, u.Name) })
}

// write runs the units as ws unit write does.
func write(t *testing.T, e kind.Env, us []kind.Unit) error {
	_, err := step.Run(t.Context(), kind.Steps(us), e.Out, false)
	return err
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

// The lines go at the top of the account's files, or root's. The env file's group unit runs a
// shell line, which TestShellEnvGroupScripts runs, so the writes leave it out.
func TestShellUnitsForAnAccountAndRoot(t *testing.T) {
	m, e := machine(t)
	m.Files["/home/zadykian/.bashrc"] = []byte("# stock\n")
	want := []string{"file/shell-bashrc-line", "file/shell-profile-line", "state/shell-env-group"}
	if got := kind.Names(units(t, e)); !slices.Equal(got, want) {
		t.Fatalf("units = %q, want %q", got, want)
	}
	us := units(t, e, want[:2]...)
	if err := write(t, e, us); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/home/zadykian/.bashrc"]); got != ". /etc/ws/bashrc\n\n# stock\n" {
		t.Errorf(".bashrc = %q, want the line at the top", got)
	}
	if got := string(m.Files["/home/zadykian/.profile"]); got != ". /etc/ws/profile\n" || m.Owners["/home/zadykian/.profile"] != [2]int{1000, 1000} {
		t.Errorf(".profile = %q, owner %v; want the line, the account's", got, m.Owners["/home/zadykian/.profile"])
	}
	for _, u := range us {
		if got := changes(t, u); got != nil {
			t.Errorf("second plan of %s = %q, want none", u.Name, got)
		}
	}
	root, err := values.Parse([]byte("WS_USER: root\n"))
	if err != nil {
		t.Fatal(err)
	}
	e.Data = root
	if err := write(t, e, units(t, e, want[:2]...)); err != nil {
		t.Fatal(err)
	}
	if string(m.Files["/root/.bashrc"]) != ". /etc/ws/bashrc\n" || string(m.Files["/root/.profile"]) != ". /etc/ws/profile\n" {
		t.Errorf("root's files = %q, %q; want the lines", m.Files["/root/.bashrc"], m.Files["/root/.profile"])
	}
}

// The env file's group unit gives the file the account's group, by id's answer, and its check
// fails on another group and passes without the file.
func TestShellEnvGroupScripts(t *testing.T) {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		State map[string]struct {
			Run struct{ Command, Check []string } `toml:"run"`
		} `toml:"state"`
	}
	if err := toml.Unmarshal(text, &spec); err != nil {
		t.Fatal(err)
	}
	r := spec.State["shell-env-group"].Run
	if len(r.Command) != 6 || len(r.Check) != 6 || r.Command[4] != envPath || r.Command[5] != "{{ .User }}" {
		t.Fatalf("command %q, check %q; want a shell line of the file and the account", r.Command, r.Check)
	}
	bin, dir := t.TempDir(), t.TempDir()
	envFile, calls := filepath.Join(dir, "env"), filepath.Join(dir, "calls")
	for name, body := range map[string]string{
		"id":    `[ "$*" = "-g -- acct" ] && echo 4242`,
		"chgrp": `echo "chgrp $*" >>` + calls,
		"stat":  `cat ` + filepath.Join(dir, "gid"),
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil { //nolint:gosec // G306: the test's own fake runs
			t.Fatal(err)
		}
	}
	group := func(argv []string) (string, bool) {
		cmd := exec.Command("sh", "-c", argv[2], argv[3], envFile, "acct") //nolint:gosec // G204: the role's own line
		cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin"}
		out, err := cmd.CombinedOutput()
		return string(out), err == nil
	}
	if out, ok := group(r.Check); !ok {
		t.Errorf("the check failed without the file: %s", out)
	}
	if out, ok := group(r.Command); !ok {
		t.Errorf("the command failed without the file: %s", out)
	}
	// A file by hand stops the command, which names it and none of its text, on the account's group too.
	for path, text := range map[string]string{envFile: "by hand, dummy\n", filepath.Join(dir, "gid"): "4242\n"} {
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := group(r.Check); ok {
		t.Error("the check passed on a file by hand")
	}
	if out, ok := group(r.Command); ok || !strings.Contains(out, envFile+": not written by ws") || strings.Contains(out, "dummy") {
		t.Errorf("the command on a file by hand: ok %v, output %q; want a failure that names the file alone", ok, out)
	}
	if _, err := os.Stat(calls); err == nil {
		t.Error("the command changed the group of a file by hand")
	}
	for name, text := range map[string]string{"env": file.Header("#") + "\n" + "export A=dummy\n", "gid": "0\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := group(r.Check); ok {
		t.Error("the check passed on root's group")
	}
	if out, ok := group(r.Command); !ok {
		t.Errorf("the command failed: %s", out)
	}
	if got, _ := os.ReadFile(calls); string(got) != "chgrp -- 4242 "+envFile+"\n" { //nolint:gosec // the test's own file
		t.Errorf("calls %q, want the account's gid set on the file", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "gid"), []byte("4242\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, ok := group(r.Check); !ok {
		t.Errorf("the check failed on the account's group: %s", out)
	}
}

func TestBashrcHoldsOneTeraMark(t *testing.T) {
	text, err := os.ReadFile("files/bashrc.tera")
	if err != nil {
		t.Fatal(err)
	}
	s := string(text)
	if strings.Count(s, "{{") != 1 || !strings.Contains(s, "export DOTNET_RUNTIME_ID=linux-{{ arch() }}\n") {
		t.Errorf("bashrc.tera's marks: want {{ arch() }} alone")
	}
	if strings.Contains(s, "{%") || strings.Contains(s, "{#") {
		t.Errorf("bashrc.tera holds a mark of Tera's that bash meant")
	}
}
