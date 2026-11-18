package kind

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// envSecrets stand in for a side's secrets: a table of names to values.
type envSecrets struct{ Env map[string]string }

// envTemplate writes the table, under ws's mark, where it holds a name.
const envTemplate = "{{ $e := (secrets).Env }}{{ if $e }}" + oursMark + "\n{{ exports $e }}{{ end -}}\n"

// envSpec is a secret env file of root's, with the account's group.
const envSpec = "[file.env]\npath = \"/etc/ws/env\"\nmode = 0o640\ntemplate = \"env.tmpl\"\nsecret = true\n" +
	"ours = true\nabsent_when_empty = true\n"

// secretEnv returns a test Env whose secrets are env, or err where set, and its machine.
func secretEnv(env map[string]string, err error) (Env, *effecttest.Machine) {
	e, m, _, _ := testEnv(false)
	m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o755, 0o755
	e.Secrets = func(context.Context) (any, error) { return envSecrets{env}, err }
	return e, m
}

func TestSecretsFuncIsASecretUnitsAlone(t *testing.T) {
	e, _ := secretEnv(map[string]string{"A": "1"}, nil)
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a\"\ntemplate = \"a.tmpl\"\n", map[string]string{"a.tmpl": envTemplate})
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), `"secrets" not defined`) {
		t.Errorf("plan = %v, want the func unknown", err)
	}
}

func TestNoSecretsLeavesTheFileAndSaysSo(t *testing.T) {
	e, m, _, out := testEnv(false)
	e.Secrets = func(context.Context) (any, error) { return nil, fmt.Errorf("the store: %w", ErrNoSecrets) }
	m.Files["/etc/ws/env"] = []byte("by hand\n")
	units := loadOne(t, e, envSpec, map[string]string{"env.tmpl": envTemplate})
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
	// A read plans the unit twice: the line comes once.
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	if want := "say: /etc/ws/env: left as it is: this machine holds no secrets file"; !slices.Equal(out.Lines, []string{want}) {
		t.Errorf("said %q, want %q once", out.Lines, want)
	}
	_, err := renderSecret(t.Context(), e, fstest.MapFS{"env.tmpl": {Data: []byte(envTemplate)}}, "env.tmpl")
	if !errors.Is(err, ErrNoSecrets) {
		t.Errorf("render = %v, want ErrNoSecrets through the template's error", err)
	}
}

func TestNilSecretsIsNoSecrets(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/ws/env"] = []byte("by hand\n")
	if got := headlines(t, loadOne(t, e, envSpec, map[string]string{"env.tmpl": envTemplate})[0].Step); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestSecretsErrorFailsThePlan(t *testing.T) {
	e, _ := secretEnv(nil, errors.New("systemd-creds can't decrypt the credential"))
	units := loadOne(t, e, envSpec, map[string]string{"env.tmpl": envTemplate})
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "can't decrypt") {
		t.Errorf("plan = %v, want the secrets' error", err)
	}
}

func TestAbsentWhenEmptyRemovesTheFile(t *testing.T) {
	e, m, _, _ := testEnv(false)
	e.Secrets = func(context.Context) (any, error) { return envSecrets{}, nil }
	m.Files["/etc/ws/env"] = []byte(oursMark + "\nexport A=hunter2\n")
	units := loadOne(t, e, envSpec, map[string]string{"env.tmpl": envTemplate})
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"remove /etc/ws/env"}) {
		t.Errorf("plan = %q, want the removal", got)
	}
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/etc/ws/env"]; ok {
		t.Error("the file stayed")
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	if _, err := Load(e, tree(map[string]string{"r/units.toml": "[file.a]\npath = \"/a\"\nsource = \"f\"\nabsent_when_empty = true\n", "r/f": ""})); err == nil ||
		!strings.Contains(err.Error(), "absent_when_empty goes with template") {
		t.Errorf("load of a source = %v, want it refused", err)
	}
}

func TestASecretFileShowsNoValue(t *testing.T) {
	e, m := secretEnv(map[string]string{"TOKEN": "hunter2"}, nil)
	units := loadOne(t, e, envSpec, map[string]string{"env.tmpl": envTemplate})
	p, err := units[0].Step.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("plan = %v, %v; want one change", p.Changes, err)
	}
	if s := p.Changes[0].String(); strings.Contains(s, "hunter2") {
		t.Errorf("change = %q, want no value", s)
	}
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if want := oursMark + "\nexport TOKEN=hunter2\n"; string(m.Files["/etc/ws/env"]) != want {
		t.Errorf("file = %q, want %q", m.Files["/etc/ws/env"], want)
	}
	bad, _ := secretEnv(map[string]string{"1A": "hunter3"}, nil)
	_, err = loadOne(t, bad, envSpec, map[string]string{"env.tmpl": envTemplate})[0].Step.Plan(t.Context())
	if err == nil || strings.Contains(err.Error(), "hunter3") || !strings.Contains(err.Error(), `"1A"`) {
		t.Errorf("plan = %v, want the name refused with no value", err)
	}
}

func TestExportsQuotesAsShlex(t *testing.T) {
	for in, want := range map[string]string{
		"": "''", "plain": "plain", "a b$x": "'a b$x'", "it's": `'it'"'"'s'`,
		"a@%+=:,./-_9": "a@%+=:,./-_9", "line\nbreak": "'line\nbreak'", "ü": "'ü'",
	} {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExportsRefusesANameAndShowsNoValue(t *testing.T) {
	if _, err := exports(map[string]string{"OK_1": "v", "_X": "v"}); err != nil {
		t.Error(err)
	}
	_, err := exports(map[string]string{"a-b": "hunter2", "1A": "hunter3", "A\n": "v", "OK": "hunter4"})
	if err == nil || strings.Contains(err.Error(), "hunter") || !strings.Contains(err.Error(), `"1A", "A\n", "a-b"`) {
		t.Errorf("got %v", err)
	}
}

func TestExportsSortsTheNames(t *testing.T) {
	got, err := exports(map[string]string{"B": "it's", "A": "plain", "C": "", "D": "a b$x"})
	want := "export A=plain\nexport B='it'\"'\"'s'\nexport C=''\nexport D='a b$x'\n"
	if err != nil || got != want {
		t.Errorf("exports = %q, %v; want %q", got, err, want)
	}
}

func TestSecretFileTakesACheck(t *testing.T) {
	e, _ := secretEnv(nil, nil)
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.env\"\nsecret = true\ncheck = [\"c\", \"{}\"]\nsource = \"f\"\n", map[string]string{"f": "TOKEN=1\n"})
	if len(units) != 1 {
		t.Fatalf("units = %d, want 1", len(units))
	}
}
