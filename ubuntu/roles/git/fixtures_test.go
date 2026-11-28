package git_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

const (
	self     = "/usr/local/bin/ws"
	check    = self + " ubuntu guest git check"
	included = self + " ubuntu guest git included"
	include  = "[include]\n\tpath = /etc/ws/gitconfig\n"
	pub      = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGit ws@box\n"
)

// role is git's spec and its templates, as kind.Load finds a role's, with the frame's, whose unit
// git's chain waits on.
func role(t *testing.T) fstest.MapFS {
	fsys := fstest.MapFS{}
	for _, name := range []string{"git/units.toml.tmpl", "git/files/ws.gitconfig.tmpl", "git/files/allowed_signers.tmpl",
		"frame/units.toml.tmpl", "frame/files/values.yml.tmpl"} {
		text, err := os.ReadFile(filepath.Join("..", name)) //nolint:gosec // a file of the roles, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys[name] = &fstest.MapFile{Data: text}
	}
	return fsys
}

// machine is a machine with root's home and /etc/ws, and its env for the values of an empty
// file, whose check passes.
func machine(t *testing.T) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/root"], m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o700, 0o755, 0o755
	m.Results[check] = effecttest.Result{}
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Self: self}
}

// loaded is the units of role's specs on e, in their order.
func loaded(t *testing.T, e kind.Env) []kind.Unit {
	all, err := kind.Load(e, role(t))
	if err != nil {
		t.Fatal(err)
	}
	return all
}

// units keeps git's units of loaded, or those named.
func units(t *testing.T, e kind.Env, names ...string) []kind.Unit {
	return slices.DeleteFunc(loaded(t, e), func(u kind.Unit) bool {
		return !strings.Contains(u.Name, "/git-") || len(names) > 0 && !slices.Contains(names, u.Name)
	})
}

// write runs the units as ws unit write does.
func write(t *testing.T, e kind.Env, us []kind.Unit) error {
	_, err := step.Run(t.Context(), kind.Steps(us), e.Out, false)
	return err
}

// apply plans the unit and applies its changes, as ws unit write does, without the check that a
// second plan holds none, which the fake's fixed answers would fail.
func apply(t *testing.T, u kind.Unit) error {
	p, err := u.Step.Plan(t.Context())
	if err != nil {
		return err
	}
	return u.Step.Apply(t.Context(), p)
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

// exits is the error of a command that exits with code, saying why on its standard error.
func exits(t *testing.T, code int, why string) error {
	_, err := exec.Command("sh", "-c", `echo "$1" >&2; exit "$2"`, "sh", why, strconv.Itoa(code)).Output() //nolint:gosec // G204: the test's own sh, its words as arguments
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("sh: %v, want an exit status", err)
	}
	return err
}

// keygen is the command line that makes the signing key, as the fake records it.
const keygen = "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " +
	"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin " + self + " ubuntu ssh keygen ws-sign"
