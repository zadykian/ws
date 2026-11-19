package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/resource/file"
)

// inputName is the form of an input's name, which is its variable's and fnox's too.
var inputName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// secretsFile is a role's mise.secrets.toml in the one form: its inputs and its files alone.
type secretsFile struct {
	Bootstrap struct {
		Secrets map[string]struct {
			Env        string `toml:"env"`
			AllowEmpty bool   `toml:"allow_empty"`
		} `toml:"secrets"`
		Files map[string]struct {
			Template    bool   `toml:"template"`
			RemoveEmpty bool   `toml:"remove_empty"`
			Owner       string `toml:"owner"`
			Mode        string `toml:"mode"`
			Content     string `toml:"content"`
		} `toml:"files"`
	} `toml:"bootstrap"`
}

// secretsFiles returns each role's mise.secrets.toml by the role's name. It decodes each strictly,
// so a key outside the form, such as a file's group or an input's short form, fails the test.
func secretsFiles(t *testing.T) map[string]secretsFile {
	paths, err := filepath.Glob(rolesDir + "/*/mise.secrets.toml")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]secretsFile{}
	for _, p := range paths {
		text, err := os.ReadFile(p) //nolint:gosec // a file of the tree, found by the glob
		if err != nil {
			t.Fatal(err)
		}
		d := toml.NewDecoder(bytes.NewReader(text))
		d.DisallowUnknownFields()
		var f secretsFile
		if err := d.Decode(&f); err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		files[filepath.Base(filepath.Dir(p))] = f
	}
	return files
}

// oursGuard returns the sh line that each secret file's template runs by mise's exec, first. It
// fails where a file is at path whose first line is not ws's header, so that a run stops before it
// replaces a file made by hand. mise's files step runs before any unit of ws's, so no unit could.
func oursGuard(path string) string {
	return "f=" + path + `; [ ! -e "$f" ] || head -n 1 -- "$f" | grep -qxF -- '` + file.Header("#") +
		`' || { echo "$f: not written by ws, so the run stops: move its value into the secrets, then remove it" >&2; exit 1; }`
}

// groupLines returns the sh lines of a secret file's group state, the command and the check, of the
// file "$1" and the account "$2". Both read the file's first line as the guard does. A run without
// the env secrets loads no guard of mise's, and stops at a file by hand here, before the account's
// group opens it.
func groupLines() (command, check string) {
	return oursGuard(`"$1"`) + `; [ ! -e "$1" ] || chgrp -- "$(id -g -- "$2")" "$1"`,
		`[ ! -e "$1" ] || { head -n 1 -- "$1" | grep -qxF -- '` + file.Header("#") +
			`' && [ "$(stat -c %g -- "$1")" = "$(id -g -- "$2")" ]; }`
}

// Each input is a table whose env is its own name, empty allowed. fnox exports a secret by the name
// it holds, and an input that a server's secrets leave empty must not stop the run.
func TestSecretInputsTakeOneForm(t *testing.T) {
	seen := map[string]string{}
	for role, f := range secretsFiles(t) {
		for name, in := range f.Bootstrap.Secrets {
			if !inputName.MatchString(name) || in.Env != name || !in.AllowEmpty {
				t.Errorf("%s: input %s has env %q, allow_empty %v; want a name of %s, env the same, allow_empty",
					role, name, in.Env, in.AllowEmpty, inputName)
			}
			if other, twice := seen[name]; twice {
				t.Errorf("%s is an input of %s and %s: one role declares it", name, other, role)
			}
			seen[name] = role
		}
	}
}

// Each file is root's at 640, a template that an empty render removes, and its content opens with
// the guard. mise takes a literal group alone, so the account's comes from a unit (below).
func TestSecretFilesTakeOneForm(t *testing.T) {
	for role, f := range secretsFiles(t) {
		for path, sf := range f.Bootstrap.Files {
			if !sf.Template || !sf.RemoveEmpty || sf.Owner != "root" || sf.Mode != "0640" {
				t.Errorf("%s: %s is template %v, remove_empty %v, owner %q, mode %q; want true, true, root, 0640",
					role, path, sf.Template, sf.RemoveEmpty, sf.Owner, sf.Mode)
			}
			first, _, _ := strings.Cut(sf.Content, "\n")
			if want := "{{- exec(command=`" + oursGuard(path) + "`) -}}"; first != want {
				t.Errorf("%s: %s's content opens with\n%s\nwant\n%s", role, path, first, want)
			}
			if strings.Count(sf.Content, "\n"+file.Header("#")+"\n") != 1 {
				t.Errorf("%s: %s's content lacks ws's header on a line of its own", role, path)
			}
		}
	}
}

// The run drops each input from a command that is not ws's, by the list.
func TestSecretInputsAreTheHiddenList(t *testing.T) {
	var got []string
	for _, f := range secretsFiles(t) {
		for name := range f.Bootstrap.Secrets {
			got = append(got, name)
		}
	}
	slices.Sort(got)
	if want := slices.Sorted(slices.Values(secretInputs)); !slices.Equal(got, want) {
		t.Errorf("the inputs of the roles' files are %q, and secretInputs %q", got, want)
	}
}

// mise writes a file root:root each time its content changes, so a unit of root's gives it the
// account's group after. Its name ends in -group, and its command and check are groupLines', of the
// path and the account.
func TestEachSecretFileHasItsGroupUnit(t *testing.T) {
	for role, f := range secretsFiles(t) {
		text, err := os.ReadFile(filepath.Join(rolesDir, role, "units.toml.tmpl")) //nolint:gosec // a role's spec
		if err != nil {
			t.Fatal(err)
		}
		var spec struct {
			State map[string]struct {
				Run struct {
					User    string   `toml:"user"`
					Command []string `toml:"command"`
					Check   []string `toml:"check"`
				} `toml:"run"`
			} `toml:"state"`
		}
		if err := toml.Unmarshal(text, &spec); err != nil {
			t.Fatalf("%s's spec: %v", role, err)
		}
		command, check := groupLines()
		for path := range f.Bootstrap.Files {
			found := false
			for name, s := range spec.State {
				r := s.Run
				found = found || strings.HasSuffix(name, "-group") && r.User == "root" &&
					slices.Equal(r.Command, []string{"sh", "-c", command, "sh", path, "{{ .User }}"}) &&
					slices.Equal(r.Check, []string{"sh", "-c", check, "sh", path, "{{ .User }}"})
			}
			if !found {
				t.Errorf("%s: no state ending in -group of root's runs groupLines on %s and the account", role, path)
			}
		}
	}
}

// The guard passes on a missing file and on ws's, and fails on one by hand, naming its path and
// none of its text.
func TestOursGuard(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	guard := func() (string, bool) {
		cmd := exec.Command("sh", "-c", oursGuard(p)) //nolint:gosec // G204: the guard's own line
		cmd.Env = []string{"PATH=/usr/bin:/bin"}
		out, err := cmd.CombinedOutput()
		return string(out), err == nil
	}
	if out, ok := guard(); !ok || out != "" {
		t.Errorf("a missing file: ok %v, output %q; want a pass in silence", ok, out)
	}
	for text, pass := range map[string]bool{
		file.Header("#") + "\ndummy-marker\n":       true,
		"by hand, dummy-marker\n":                   false,
		"x" + file.Header("#") + "\ndummy-marker\n": false,
	} {
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		out, ok := guard()
		if ok != pass || strings.Contains(out, "dummy-marker") || !pass && !strings.Contains(out, p+": not written by ws") {
			t.Errorf("a file of %q: ok %v, output %q; want ok %v, its path named and none of its text", text, ok, out, pass)
		}
	}
}
