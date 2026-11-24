package otelcol_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

const (
	envPath  = "/etc/ws/env"
	envFile  = "/etc/otelcol-contrib/claude.env"
	config   = "/etc/otelcol-contrib/config.yaml"
	variable = "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT"
	dropIns  = "/etc/systemd/system/otelcol-contrib.service.d/"
)

// specText reads a file of the role, beside this test.
func specText(t *testing.T, name string) []byte {
	text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named by the test
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// endpointCheck returns the check of claude.env's part in units.toml.tmpl: the command that refuses a
// bad endpoint, {} standing for the file.
func endpointCheck(t *testing.T) []string { return checkOf(t, envFile) }

// checkOf returns the check of a part of the service, by its path.
func checkOf(t *testing.T, path string) []string {
	var spec struct {
		Service map[string]struct {
			File []struct {
				Path  string   `toml:"path"`
				Check []string `toml:"check"`
			} `toml:"file"`
		} `toml:"service"`
	}
	if err := toml.Unmarshal(specText(t, "units.toml.tmpl"), &spec); err != nil {
		t.Fatal(err)
	}
	for _, f := range spec.Service["otelcol"].File {
		if f.Path == path {
			return f.Check
		}
	}
	t.Fatalf("units.toml.tmpl has no part for %s", path)
	return nil
}

func TestEndpointCheck(t *testing.T) {
	check := endpointCheck(t)
	dir := t.TempDir()
	for _, c := range []struct {
		url string
		ok  bool
	}{
		{"http://collector.example:4317", true},
		{"https://10.0.0.5:4318", true},
		{"https://[fd00::1]:4317", true},
		{"http://host_name:4317", true},
		{"collector:4317", false},
		{"http://collector", false},
		{"http://collector:0", false},
		{"http://collector:65535", true},
		{"http://collector:65536", false},
		{"http://collector:99999", false},
		{"http://collector:100000", false},
		{"http://collector:4317/path", false},
		{"HTTP://collector:4317", false},
		{"http://127.0.0.1:4317", false},
		{"http://127.0.0.1:4318", true},
		{"http://0.0.0.0:4317", false},
		{"http://[::]:4317", false},
		{"http://[0:0::0]:4317", false},
		{"http://[::ffff:127.0.0.1]:4317", false},
		{"http://localhost:4317", false},
		{"https://LocalHost.:4317", false},
		{"http://a.localhost:4317", false},
		{"http://notlocalhost:4317", true},
	} {
		path := filepath.Join(dir, "claude.env")
		if err := os.WriteFile(path, []byte("# header\n"+variable+"="+c.url+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		args := slices.Clone(check[1:])
		args[len(args)-1] = path
		cmd := exec.Command(check[0], args...) //nolint:gosec // the role's own check, run on a file of the test
		if err := cmd.Run(); (err == nil) != c.ok {
			t.Errorf("check of %q = %v, want ok %v", c.url, err, c.ok)
		}
	}
}

// checkLine is the command line of a part's check, as the machine sees it: {} is the staged file.
func checkLine(t *testing.T, path string) string {
	check := slices.Clone(checkOf(t, path))
	for i, a := range check {
		check[i] = strings.ReplaceAll(a, "{}", path+".ws-check")
	}
	return strings.Join(check, " ")
}

// machine is a machine with the collector's directories and /etc/ws/env set to env, and
// the environment that loads the role's units over it.
func machine(t *testing.T, env string) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	for _, d := range []string{"/etc", "/etc/ws", "/etc/systemd", "/etc/systemd/system", "/etc/otelcol-contrib"} {
		m.Dirs[d] = 0o755
	}
	if env != "" {
		m.Files[envPath] = []byte(env)
	}
	for _, p := range []string{envFile, config} {
		m.Results[checkLine(t, p)] = effecttest.Result{}
	}
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v}
}

// collector loads otelcol's units, with a stand-in for signoz's sync, which the .deb waits on, and
// returns the collector's service unit.
func collector(t *testing.T, e kind.Env) []kind.Unit {
	fsys := fstest.MapFS{
		"signoz/units.toml.tmpl": {Data: []byte("[state.signoz-sync.run]\nuser = \"root\"\ncommand = [\"true\"]\ncreates = [\"/etc\"]\n")},
	}
	for _, name := range []string{"files/journal.conf", "files/claude.conf", "files/claude.env.tmpl", "files/config.yaml.tmpl"} {
		fsys["otelcol/"+name] = &fstest.MapFile{Data: specText(t, name)}
	}
	// The .deb's install is apt's, which the test leaves out: the service waits on nothing here.
	spec := strings.Replace(string(specText(t, "units.toml.tmpl")), "after = \"deb/otelcol\"\n", "", 1)
	fsys["otelcol/units.toml.tmpl"] = &fstest.MapFile{Data: []byte(spec)}
	all, err := kind.Load(e, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return slices.DeleteFunc(all, func(u kind.Unit) bool { return u.Name != "service/otelcol" })
}

func TestCollectorFilesFollowTheEndpoint(t *testing.T) {
	for _, c := range []struct {
		env, endpoint   string
		remote, tlsFree bool
	}{
		{"", "", false, false},
		{"OTHER=1\n", "", false, false},
		{variable + "=https://c.example:4317\n", "https://c.example:4317", true, false},
		{"export " + variable + "='http://c.example:4317'\n", "http://c.example:4317", true, true},
	} {
		m, e := machine(t, c.env)
		units := collector(t, e)
		if _, err := step.Run(t.Context(), kind.Steps(units), e.Out, false); err != nil {
			t.Fatalf("env %q: %v", c.env, err)
		}
		config := m.Files[config]
		var parsed map[string]any
		if err := yaml.Unmarshal(config, &parsed); err != nil {
			t.Fatalf("env %q: the config is no YAML: %v", c.env, err)
		}
		if got := strings.Contains(string(config), "metrics/claude:"); got != c.remote {
			t.Errorf("env %q: claude's pipeline = %v, want %v", c.env, got, c.remote)
		}
		claude := exporter(parsed, "otlp_grpc/claude")
		if got := claude["tls"] != nil; got != c.tlsFree {
			t.Errorf("env %q: claude's exporter has tls %v, want %v", c.env, claude["tls"], c.tlsFree)
		}
		if c.endpoint != "" && strings.Contains(string(config), c.endpoint) {
			t.Errorf("env %q: the config holds the endpoint", c.env)
		}
		got, ok := m.Files[envFile]
		if want := c.endpoint != ""; ok != want {
			t.Fatalf("env %q: claude.env present = %v, want %v", c.env, ok, want)
		}
		if ok && (!strings.Contains(string(got), "\n"+variable+"="+c.endpoint+"\n") || m.Modes[envFile] != 0o600) {
			t.Errorf("env %q: claude.env = %q, mode %o; want the line, 0600", c.env, got, m.Modes[envFile])
		}
		for _, name := range []string{"journal.conf", "claude.conf"} {
			if len(m.Files[dropIns+name]) == 0 {
				t.Errorf("env %q: %s is not written", c.env, name)
			}
		}
	}
}

func TestEndpointGoneRemovesClaudeEnv(t *testing.T) {
	m, e := machine(t, variable+"=https://c.example:4317\n")
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	delete(m.Files, envPath)
	units := collector(t, e)
	if _, err := step.Run(t.Context(), kind.Steps(units), e.Out, false); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files[envFile]; ok {
		t.Error("claude.env stayed where no endpoint is set")
	}
	if strings.Contains(string(m.Files[config]), "metrics/claude:") {
		t.Error("the config keeps claude's pipeline")
	}
}

// The config that names the endpoint's variable goes before claude.env, which sets it: a refused
// config keeps both.
func TestRefusedConfigKeepsClaudeEnv(t *testing.T) {
	m, e := machine(t, variable+"=https://c.example:4317\n")
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	live := string(m.Files[config])
	delete(m.Files, envPath)
	m.Results[checkLine(t, config)] = effecttest.Result{Err: errors.New("invalid config")}
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err == nil {
		t.Fatal("a config the validator refuses was applied")
	}
	if _, ok := m.Files[envFile]; !ok || string(m.Files[config]) != live {
		t.Errorf("claude.env there = %v, config changed = %v; want claude.env and the live config", ok, string(m.Files[config]) != live)
	}
}

// restarts counts the collector's restarts in the machine's calls.
func restarts(u *effecttest.Units) int {
	n := 0
	for _, c := range u.Calls {
		if strings.Contains(c, "restart") && strings.Contains(c, "otelcol-contrib.service") {
			n++
		}
	}
	return n
}

func TestTwoFilesChangeOneRestart(t *testing.T) {
	m, e := machine(t, variable+"=https://c.example:4317\n")
	u := e.U.(*effecttest.Units)
	u.States = map[string]effect.UnitState{"otelcol-contrib.service": {Enabled: true, Active: true}}
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	before := restarts(u)
	m.Files[envPath] = []byte(variable + "=https://d.example:4317\n")
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	if got := restarts(u) - before; got != 1 {
		t.Errorf("restarts after the endpoint changed (two files) = %d, want 1", got)
	}
}

func TestRefusedConfigLeavesTheLiveFile(t *testing.T) {
	m, e := machine(t, "")
	u := e.U.(*effecttest.Units)
	u.States = map[string]effect.UnitState{"otelcol-contrib.service": {Enabled: true, Active: true}}
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	live := string(m.Files[config])
	before := restarts(u)
	m.Files[envPath] = []byte(variable + "=https://c.example:4317\n")
	m.Results[checkLine(t, config)] = effecttest.Result{Err: errors.New("invalid config")}
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err == nil {
		t.Fatal("a config the validator refuses was applied")
	}
	if string(m.Files[config]) != live {
		t.Error("the live config changed")
	}
	if got := restarts(u) - before; got != 0 {
		t.Errorf("restarts after a refusal = %d, want 0", got)
	}
}
