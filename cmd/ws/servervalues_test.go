package main

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestWriteVarsStartsTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	if err := writeVars(m, []byte("---\n")); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[serverVars]); got != "---\n" {
		t.Errorf("file = %q", got)
	}
	if m.Dirs["/etc/ws"] != 0o755 || m.Modes[serverVars] != 0o600 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[serverVars])
	}
}

func TestWriteVarsKeepsTheModeOfAFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/ws"] = 0o700
	m.Files[serverVars] = []byte("---\n")
	m.Modes[serverVars] = 0o640
	if err := writeVars(m, []byte("---\nws_user: ada\n")); err != nil {
		t.Fatal(err)
	}
	if m.Dirs["/etc/ws"] != 0o700 || m.Modes[serverVars] != 0o640 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[serverVars])
	}
}

// valuesProcess returns a root process on a machine that holds file as the values.
func valuesProcess(file string) (process, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.machine, p.euid = m, 0
	m.Dirs["/etc/ws"] = 0o755
	m.Files[serverVars] = []byte(file)
	return p, m
}

func TestValuesSetWritesOneKey(t *testing.T) {
	p, m := valuesProcess("---\n# the machine\nws_user: ada\n")
	status, stdout, stderr := runWs(p, "", "server", "values", "set", "base_swappiness", "20")
	if status != 0 || stdout != "" || !strings.Contains(stderr, "wrote "+serverVars) {
		t.Fatalf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if got := string(m.Files[serverVars]); got != "---\n# the machine\nws_user: ada\nbase_swappiness: 20\n" {
		t.Errorf("file = %q", got)
	}
}
