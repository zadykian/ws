package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/values"
)

func TestValuesCheckPrintsOneLineForFilesThatPass(t *testing.T) {
	m := effecttest.NewMachine()
	writeConfs(m, values.Text("WS_SLOT", "0"))
	p := withEnv(fakeProcess(), map[string]string{"WS_VALUES": ""})
	p.Machine = m
	status, stdout, stderr := runWs(p, "macos", "values", "check")
	if status != 0 || stdout != "ws values: checked\n" || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0, the line of a pass, no error", status, stdout, stderr)
	}
}

func TestValuesCheckNamesEachBadKeyWithItsFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/bad/values.yml"] = []byte(values.Text("WS_TARGET", "lan"))
	m.Files["/bad/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "999"))
	p := withEnv(fakeProcess(), map[string]string{"WS_VALUES": "/bad/values.yml"})
	p.Machine = withServerDirs(m)
	status, _, stderr := runWs(p, "macos", "values", "check")
	if status != 1 {
		t.Errorf("status = %d, want 1", status)
	}
	for _, want := range []string{
		"/bad/values.yml, line 1: WS_TARGET is a server's key",
		"/bad/servers/ws/values.yml, line 1: WS_SLOT=999 is not a number from 0 to",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to contain %q", stderr, want)
		}
	}
}

func TestValuesCheckRefusesAServerFlag(t *testing.T) {
	m := effecttest.NewMachine()
	writeConfs(m, values.Text("WS_SLOT", "0"))
	p := withEnv(fakeProcess(), map[string]string{"WS_VALUES": ""})
	p.Machine = m
	status, stdout, stderr := runWs(p, "macos", "--server", "ws", "values", "check")
	if want := "values check checks every server: leave out --server\n"; status != 1 || stdout != "" || !strings.HasSuffix(stderr, want) {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 1, none, an error ending %q", status, stdout, stderr, want)
	}
}
