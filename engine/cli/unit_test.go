package cli_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// unitValues stand in for a side's values in the specs and their templates.
type unitValues struct{ User string }

// unitEnv returns a root's Env whose WS_UNITS holds the specs, by their paths below it, and the
// fake machine that the units run on, with /etc.
func unitEnv(t *testing.T, specs map[string]string) (cli.Env, *effecttest.Machine) {
	dir := t.TempDir()
	for name, text := range specs {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	m := effecttest.NewMachine()
	m.Dirs["/etc"] = 0o755
	e := clitest.WithEnv(clitest.Env(), map[string]string{"WS_UNITS": dir})
	// ws runs in the specs' parent, as the plugin's hooks do in mise's run directory.
	e.Machine, e.Euid, e.Pwd = m, 0, filepath.Dir(dir)
	return e, m
}

// runUnit runs ws unit with args on a side of fixed values, and returns the status, stdout and
// stderr.
func runUnit(e cli.Env, args ...string) (status int, stdout, stderr string) {
	side := cli.UnitSide{
		Dir: "/nowhere",
		Values: func(context.Context, effect.Machine) (any, error) {
			return unitValues{User: "u"}, nil
		},
		Units: &effecttest.Units{},
		Root:  true,
	}
	return clitest.Run(e, []*cobra.Command{cli.Unit(e, side)}, "", append([]string{"unit"}, args...)...)
}

// fileUnit is the spec of a file unit at path with the source name, after the unit after where
// that is not empty.
func fileUnit(name, path, after string) string {
	spec := "[file." + name + "]\npath = \"" + path + "\"\nsource = \"" + name + "\"\n"
	if after != "" {
		spec += "after = \"" + after + "\"\n"
	}
	return spec
}

func TestUnitListRunOrder(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": fileUnit("a", "/etc/a", "file/b") + fileUnit("b", "/etc/b", ""),
		"s/units.toml.tmpl": fileUnit("c", "/etc/c", ""),
		"r/a":               "1\n", "r/b": "2\n", "s/c": "3\n",
	})
	e.Euid = 1000
	if status, stdout, stderr := runUnit(e, "list"); status != 0 || stdout != "file/b\nfile/a\nfile/c\n" {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestUnitListTOML(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "1\n"})
	if status, stdout, _ := runUnit(e, "list", "--toml"); status != 0 || stdout != "\"ws:file/a\" = \"in-sync\"\n" {
		t.Errorf("status = %d, stdout = %q", status, stdout)
	}
}

func TestUnitListWithoutSpecs(t *testing.T) {
	e, _ := unitEnv(t, nil)
	if status, stdout, stderr := runUnit(e, "list"); status != 0 || stdout != "" || stderr != "" {
		t.Errorf("an empty root: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	e = clitest.WithEnv(e, map[string]string{"WS_UNITS": filepath.Join(t.TempDir(), "gone")})
	if status, _, stderr := runUnit(e, "list"); status != 1 || !strings.Contains(stderr, "the specs:") {
		t.Errorf("a missing root: status = %d, stderr = %q", status, stderr)
	}
}

func TestUnitListFailsOnABadSpec(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{"r/units.toml.tmpl": "[file.a]\npth = \"/etc/a\"\n"})
	if status, stdout, stderr := runUnit(e, "list"); status != 1 || stdout != "" || !strings.Contains(stderr, "pth") {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestUnitReadStates(t *testing.T) {
	e, m := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": fileUnit("same", "/etc/same", "") + fileUnit("new", "/etc/x/new", "") +
			fileUnit("behind", "/etc/behind", "file/new") +
			"[file.bad]\npath = \"/etc/bad\"\ntemplate = \"bad.tmpl\"\n",
		"r/same": "1\n", "r/new": "2\n", "r/behind": "3\n", "r/bad.tmpl": "{{.Missing}}\n",
	})
	m.Files["/etc/same"] = []byte("1\n")
	status, stdout, stderr := runUnit(e, "read", "file/bad", "file/behind", "file/same", "file/new")
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if status != 0 || len(lines) != 4 || !strings.HasPrefix(lines[0], "file/bad\tfails: ") ||
		!strings.Contains(lines[0], "Missing") {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if lines[1] != "file/behind\tafter file/new" || lines[2] != "file/same\tin-sync" ||
		!strings.HasPrefix(lines[3], "file/new\t") || !strings.HasSuffix(lines[3], " (+1 more)") {
		t.Errorf("states = %q", lines[1:])
	}
	if _, wrote := m.Files["/etc/x/new"]; wrote {
		t.Error("read wrote file/new")
	}
}

func TestUnitUnknownUnitIsUsage(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "1\n"})
	for _, verb := range []string{"read", "write"} {
		if status, _, stderr := runUnit(e, verb, "file/z"); status != 2 || !strings.Contains(stderr, `no unit "file/z"`) {
			t.Errorf("%s: status = %d, stderr = %q", verb, status, stderr)
		}
	}
}

func TestUnitWriteAppliesAndPlansAgain(t *testing.T) {
	e, m := unitEnv(t, map[string]string{"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "1\n"})
	status, _, stderr := runUnit(e, "write", "file/a")
	if status != 0 || string(m.Files["/etc/a"]) != "1\n" || !strings.Contains(stderr, "changed: file/a\n") {
		t.Fatalf("status = %d, file = %q, stderr = %q", status, m.Files["/etc/a"], stderr)
	}
	if status, stdout, _ := runUnit(e, "read", "file/a"); status != 0 || stdout != "file/a\tin-sync\n" {
		t.Errorf("read after write: status = %d, stdout = %q", status, stdout)
	}
	if status, _, stderr := runUnit(e, "write", "file/a"); status != 0 || !strings.Contains(stderr, "changed: none\n") {
		t.Errorf("second write: status = %d, stderr = %q", status, stderr)
	}
}

func TestUnitWriteLeavesOutTheUnnamed(t *testing.T) {
	e, m := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": fileUnit("a", "/etc/a", "file/b") + fileUnit("b", "/etc/b", "") + fileUnit("c", "/etc/c", ""),
		"r/a":               "1\n", "r/b": "2\n", "r/c": "3\n",
	})
	m.Files["/etc/b"] = []byte("2\n")
	if status, _, stderr := runUnit(e, "write", "file/a"); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if string(m.Files["/etc/a"]) != "1\n" || m.Files["/etc/c"] != nil {
		t.Errorf("files = %q, %q; want a written and c left", m.Files["/etc/a"], m.Files["/etc/c"])
	}
}

func TestUnitWriteSkipsBehindAChange(t *testing.T) {
	e, m := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": fileUnit("a", "/etc/a", "file/b") + fileUnit("b", "/etc/b", ""),
		"r/a":               "1\n", "r/b": "2\n",
	})
	status, _, stderr := runUnit(e, "write", "file/a")
	if status != 1 || !strings.Contains(stderr, "skipped file/a (after file/b)") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if m.Files["/etc/a"] != nil || m.Files["/etc/b"] != nil {
		t.Errorf("files = %q, %q; want neither written", m.Files["/etc/a"], m.Files["/etc/b"])
	}
}

func TestUnitWriteDryRunAppliesNothing(t *testing.T) {
	e, m := unitEnv(t, map[string]string{"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "new\n"})
	m.Files["/etc/a"] = []byte("old\n")
	status, stdout, stderr := runUnit(e, "write", "--dry-run", "file/a")
	if status != 0 || stdout != "" || !strings.Contains(stderr, "-old\n+new\n") ||
		!strings.Contains(stderr, "would change: file/a\n") {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if got := string(m.Files["/etc/a"]); got != "old\n" {
		t.Errorf("file = %q, want the old one", got)
	}
}

func TestUnitWriteRefusedCheckIsOne(t *testing.T) {
	e, m := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": "[file.a]\npath = \"/etc/a\"\nsource = \"a\"\ncheck = [\"checker\", \"{}\"]\n",
		"r/a":               "new\n",
	})
	m.Files["/etc/a"] = []byte("old\n")
	m.Results["checker /etc/a.ws-check"] = effecttest.Result{Err: errors.New("exit status 1")}
	status, _, stderr := runUnit(e, "write", "file/a")
	if status != 1 || !strings.Contains(stderr, "checker refuses the new /etc/a") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if got := string(m.Files["/etc/a"]); got != "old\n" {
		t.Errorf("file = %q, want the live one kept", got)
	}
}

func TestUnitReadOffUnitIsInSync(t *testing.T) {
	e, m := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": fileUnit("a", "/etc/a", "") + "when = { board_vendor = \"Acme Ltd.\" }\n",
		"r/a":               "1\n",
	})
	status, stdout, stderr := runUnit(e, "read", "file/a")
	if status != 0 || stdout != "file/a\tin-sync\n" || strings.Count(stderr, "file/a: off on this machine") != 1 {
		t.Errorf("read: status = %d, stdout = %q, stderr = %q; want the off line once", status, stdout, stderr)
	}
	status, _, stderr = runUnit(e, "write", "file/a")
	if _, wrote := m.Files["/etc/a"]; status != 0 || wrote || !strings.HasSuffix(stderr, "changed: none\n") {
		t.Errorf("write: status = %d, wrote %v, stderr = %q", status, wrote, stderr)
	}
}

// offUnit is the spec of file/a at /etc/a, off on a machine without the board of Acme Ltd.
const offUnit = "[file.a]\npath = \"/etc/a\"\nsource = \"a\"\nwhen = { board_vendor = \"Acme Ltd.\" }\n"

func TestUnitDryRunOffUnitChangesNothing(t *testing.T) {
	e, m := unitEnv(t, map[string]string{"r/units.toml.tmpl": offUnit, "r/a": "1\n"})
	status, _, stderr := runUnit(e, "write", "--dry-run", "file/a")
	if _, wrote := m.Files["/etc/a"]; status != 0 || wrote || strings.Count(stderr, "file/a: off on this machine") != 1 {
		t.Errorf("dry run: status = %d, wrote %v, stderr = %q; want the off line once, and no file", status, wrote, stderr)
	}
}

func TestUnitWritesAUnitAfterAnOffOne(t *testing.T) {
	e, m := unitEnv(t, map[string]string{"r/units.toml.tmpl": offUnit + fileUnit("b", "/etc/b", "file/a"), "r/a": "1\n", "r/b": "2\n"})
	status, _, stderr := runUnit(e, "write", "file/a", "file/b")
	if _, wrote := m.Files["/etc/a"]; status != 0 || wrote || string(m.Files["/etc/b"]) != "2\n" ||
		!strings.HasSuffix(stderr, "changed: file/b\n") {
		t.Errorf("write: status = %d, wrote a %v, b = %q, stderr = %q; want b alone", status, wrote, m.Files["/etc/b"], stderr)
	}
}

func TestUnitSetsSelf(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{"r/units.toml.tmpl": "[file.a]\npath = \"/etc{{ self }}\"\nsource = \"a\"\n", "r/a": "1\n"})
	status, _, stderr := runUnit(e, "write", "--dry-run", "file/a")
	if status != 0 || !strings.Contains(stderr, "/etc/h/.local/bin/ws") {
		t.Errorf("status = %d, stderr = %q; want the path of this ws", status, stderr)
	}
	e.Self = func() (string, error) { return "", errors.New("no such file") }
	if status, _, stderr := runUnit(e, "list"); status != 1 || !strings.Contains(stderr, "finding the file ws runs from") {
		t.Errorf("without the path: status = %d, stderr = %q", status, stderr)
	}
}

func TestUnitReadWholeNamesALackingLine(t *testing.T) {
	e, m := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": fileUnit("a", "/etc/a", "") + fileUnit("b", "/etc/b", "") + fileUnit("c", "/etc/c", ""),
		"r/a":               "1\n", "r/b": "2\n", "r/c": "3\n",
	})
	// The run directory is the specs' parent, which the failure names too.
	hint := ", or run mise in " + filepath.Dir(e.Getenv("WS_UNITS")) + "\n"
	status, stdout, stderr := runUnit(e, "read", "--whole", "file/a")
	want := "file/a\tfails: mise's plan lacks ws:file/b, ws:file/c: add each to [bootstrap.packages]" + hint
	if status != 0 || stdout != want {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want %q", status, stdout, stderr, want)
	}
	// mise reads again before it installs, without the lines in sync: b, once written, fails no read.
	if status, _, stderr := runUnit(e, "write", "file/b"); status != 0 || string(m.Files["/etc/b"]) != "2\n" {
		t.Fatalf("write b: status = %d, stderr = %q", status, stderr)
	}
	status, stdout, _ = runUnit(e, "read", "--whole", "file/a")
	if want := "file/a\tfails: mise's plan lacks ws:file/c: add each to [bootstrap.packages]" + hint; status != 0 || stdout != want {
		t.Errorf("with b in sync: status = %d, stdout = %q; want %q", status, stdout, want)
	}
	if status, stdout, _ := runUnit(e, "read", "--whole", "file/c", "file/b", "file/a"); status != 0 ||
		strings.Contains(stdout, "fails") || strings.Count(stdout, "\n") != 3 {
		t.Errorf("every line: status = %d, stdout = %q; want three states, none failed", status, stdout)
	}
	if status, stdout, _ := runUnit(e, "read", "file/a"); status != 0 || strings.Contains(stdout, "fails") {
		t.Errorf("without --whole: status = %d, stdout = %q; want the unit read alone", status, stdout)
	}
	// Outside the run directory mise loads none of the specs' lines, so each line fails, in sync too.
	runDir := e.Pwd
	e.Pwd = e.Getenv("WS_UNITS")
	want = "file/c\tfails: mise runs in " + e.Pwd + ", which loads none of the specs' lines: run mise in " + runDir + "\n" +
		"file/b\tfails: mise runs in " + e.Pwd + ", which loads none of the specs' lines: run mise in " + runDir + "\n"
	if status, stdout, _ := runUnit(e, "read", "--whole", "file/c", "file/b"); status != 0 || stdout != want {
		t.Errorf("outside the run directory: status = %d, stdout = %q; want %q", status, stdout, want)
	}
}
