package kind

import (
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

func TestFactsFailWhereTheMachineLacksThem(t *testing.T) {
	for _, text := range []string{`{{mem_mib}}`, `{{cpus}}`, `{{fs_bytes "/none"}}`, `{{uid "nobody"}}`, `{{div 1 0}}`} {
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
