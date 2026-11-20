package kind

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// factsMachine is a machine of 2 processors, 4 GiB and a filesystem of 10 blocks of 4096 bytes at
// /var, with an account u of uid 1001.
func factsMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/proc/meminfo"] = []byte("MemTotal:        4194304 kB\nMemFree:          100 kB\n")
	m.Files["/proc/cpuinfo"] = []byte("processor\t: 0\nmodel name\t: x\n\nprocessor\t: 1\nmodel name\t: x\n")
	m.Dirs["/var"] = 0o755
	m.Results["stat -f -c %S %b /var"] = effecttest.Result{Out: []byte("4096 10\n")}
	m.Results["id -u u"] = effecttest.Result{Out: []byte("1001\n")}
	m.Results["id -g u"] = effecttest.Result{Out: []byte("1001\n")}
	return m
}

func TestFactsReadTheMachine(t *testing.T) {
	m := factsMachine()
	m.Files["/etc/ws/env"] = []byte("A=1\nexport A=\"two\"\nB='three'\n")
	dir := fstest.MapFS{"t.tmpl": {Data: []byte(
		`{{mem_mib}} {{cpus}} {{fs_bytes "/var/swap" "/var"}} {{uid "u"}} {{uid "root"}} ` +
			`{{envvar "/etc/ws/env" "A"}} {{envvar "/etc/ws/env" "B"}} [{{envvar "/etc/ws/none" "A"}}] ` +
			`{{div (mul (add 1 3) 5) 2}}`)}}
	got, err := render(t.Context(), Env{M: m}, dir, "t.tmpl")
	if want := "4096 2 40960 1001 0 two three [] 10"; err != nil || string(got) != want {
		t.Errorf("render = %q, %v; want %q", got, err, want)
	}
}

// A variable that the side hides is an input, which mise writes to the env file in the same run.
// Where the run's env sets it, envvar takes it from there, so a plan sees what the run writes.
func TestEnvvarTakesAnInputOfTheRun(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/ws/env"] = []byte("export A='old'\nB=old\n")
	dir := fstest.MapFS{"t.tmpl": {Data: []byte(`{{envvar "/etc/ws/env" "A"}}|{{envvar "/etc/ws/env" "B"}}`)}}
	for _, c := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"A": "new", "B": "new"}, "new|old"},
		{map[string]string{"A": ""}, "|old"},
		{map[string]string{}, "old|old"},
		{nil, "old|old"},
	} {
		e := Env{M: m, Hidden: []string{"A"}}
		if c.env != nil {
			e.LookupEnv = func(k string) (string, bool) {
				v, ok := c.env[k]
				return v, ok
			}
		}
		if got, err := render(t.Context(), e, dir, "t.tmpl"); err != nil || string(got) != c.want {
			t.Errorf("env %v: render = %q, %v; want %q", c.env, got, err, c.want)
		}
	}
}

func TestRootFileReadsAsRoot(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/k/a.pub"] = []byte("  key\n")
	dir := fstest.MapFS{"t.tmpl": {Data: []byte(`[{{root_file "/etc/k/a.pub"}}]`)}}
	got, err := render(t.Context(), Env{M: m}, dir, "t.tmpl")
	if err != nil || string(got) != "[key]" {
		t.Errorf("render = %q, %v; want [key]", got, err)
	}
	if !slices.Contains(m.Calls, "read root /etc/k/a.pub") {
		t.Errorf("calls %q, want the file read as root", m.Calls)
	}
}

func TestRootFileMissingIsEmpty(t *testing.T) {
	dir := fstest.MapFS{"t.tmpl": {Data: []byte(`[{{with root_file "/etc/k/b.pub"}}x{{end}}]`)}}
	got, err := render(t.Context(), Env{M: effecttest.NewMachine()}, dir, "t.tmpl")
	if err != nil || string(got) != "[]" {
		t.Errorf("render = %q, %v; want []", got, err)
	}
}

func TestFactsFailWhereTheMachineLacksThem(t *testing.T) {
	for _, text := range []string{`{{mem_mib}}`, `{{cpus}}`, `{{fs_bytes "/none"}}`, `{{uid "nobody"}}`, `{{div 1 0}}`,
		`{{installed_version "a"}}`} {
		dir := fstest.MapFS{"t.tmpl": {Data: []byte(text)}}
		if got, err := render(t.Context(), Env{M: effecttest.NewMachine()}, dir, "t.tmpl"); err == nil {
			t.Errorf("render %s = %q, want an error", text, got)
		}
	}
}

func TestRenderFailsOnAMissingKey(t *testing.T) {
	type values struct{ User string }
	for _, c := range []struct {
		data any
		text string
	}{
		{map[string]string{"User": "u"}, "{{.Home}}"},
		{values{User: "u"}, "{{.Home}}"},
	} {
		got, err := fill("t", []byte(c.text), pure(), c.data)
		if err == nil || !strings.Contains(err.Error(), "Home") {
			t.Errorf("fill %s over %T = %q, %v; want an error that names Home", c.text, c.data, got, err)
		}
	}
	if got, err := fill("t", []byte("{{.User}}"), pure(), values{User: "u"}); err != nil || string(got) != "u" {
		t.Errorf("fill = %q, %v; want u", got, err)
	}
}

func TestSpecFuncsReadNothing(t *testing.T) {
	if _, err := fill("t", []byte("{{mem_mib}}"), pure(), nil); err == nil {
		t.Error("a spec's text took mem_mib, which reads the machine before any unit ran")
	}
}

func TestJSONQuotesAsGosEncoder(t *testing.T) {
	got, err := fill("t", []byte(`{{ json "a\"b<" }} {{ json 3 }} {{ json .User }}`), pure(), map[string]string{"User": "u&"})
	if want := `"a\"b\u003c" 3 "u\u0026"`; err != nil || string(got) != want {
		t.Errorf("fill = %s, %v; want %s", got, err, want)
	}
}

func TestInstalledVersion(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"a": {Version: "1.2-3", Status: "ii"}, "b": {Version: "4.5", Status: "rc"}}
	dir := fstest.MapFS{"t.tmpl": {Data: []byte(`{{installed_version "a"}}|{{installed_version "b"}}|{{installed_version "c"}}`)}}
	if got, err := render(t.Context(), Env{M: m}, dir, "t.tmpl"); err != nil || string(got) != "1.2-3||" {
		t.Errorf("render = %q, %v; want 1.2-3, then none for b and c", got, err)
	}
}

func TestFailStopsTheRender(t *testing.T) {
	got, err := fill("t", []byte(`a{{if true}}{{fail "no kernel fits"}}{{end}}`), pure(), nil)
	if err == nil || !strings.Contains(err.Error(), "no kernel fits") {
		t.Errorf("fill = %q, %v; want the render stopped with fail's message", got, err)
	}
}

func TestKernelRelease(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["uname -r"] = effecttest.Result{Out: []byte("7.0.0-1-generic\n")}
	dir := fstest.MapFS{"t.tmpl": {Data: []byte(`linux-headers-{{kernel_release}}`)}}
	if got, err := render(t.Context(), Env{M: m}, dir, "t.tmpl"); err != nil || string(got) != "linux-headers-7.0.0-1-generic" {
		t.Errorf("render = %q, %v; want the release, trimmed", got, err)
	}
	if got, err := render(t.Context(), Env{M: effecttest.NewMachine()}, dir, "t.tmpl"); err == nil {
		t.Errorf("render without uname = %q, want an error", got)
	}
}

func TestHasPrefix(t *testing.T) {
	got, err := fill("t", []byte(`{{ if has_prefix "http://a" "http://" }}yes{{ end }}{{ if has_prefix "https://a" "http://" }}no{{ end }}`), pure(), nil)
	if err != nil || string(got) != "yes" {
		t.Errorf("fill = %q, %v; want yes", got, err)
	}
}

func TestTrimSuffix(t *testing.T) {
	got, err := fill("t", []byte(`{{ trim_suffix "/h/.ssh/k.pub" ".pub" }} {{ trim_suffix "/h/k" ".pub" }}`), pure(), nil)
	if want := "/h/.ssh/k /h/k"; err != nil || string(got) != want {
		t.Errorf("fill = %q, %v; want %q", got, err, want)
	}
}
