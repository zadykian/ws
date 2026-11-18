package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/values"
)

func TestSetupCommand(t *testing.T) {
	m := effecttest.NewMachine()
	writeConfs(m, "", values.Text("WS_USER", "ada"))
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = m
	status, _, stderr := runWs(p, "client", "macos", "apply", "--bogus")
	if status != 2 || !strings.Contains(stderr, "ws client macos apply: unknown flag: --bogus") {
		t.Errorf("status = %d, stderr = %q; want a usage error", status, stderr)
	}
	p = fakeProcess()
	p.Machine = effecttest.NewMachine()
	if status, _, stderr = runWs(p, "client", "macos", "apply"); status != 1 || !strings.Contains(stderr, "no server in") {
		t.Errorf("with no file: status = %d, stderr = %q; want no server", status, stderr)
	}
}

func TestSetupRefusesRoot(t *testing.T) {
	p := fakeProcess()
	p.Euid = 0
	if status, _, stderr := runWs(p, "client", "macos", "apply"); status != 1 ||
		!strings.HasPrefix(stderr, "ws client macos apply: run it as yourself, not root") {
		t.Errorf("as root: status = %d, stderr = %q", status, stderr)
	}
}

func TestMacChangeSaysADiffBareAndNoBackup(t *testing.T) {
	out := &effecttest.Out{}
	o := linesOf(out, "ws client macos apply")
	plan := []step.Change{file.UserWrite{Path: "/f", After: []byte("a\n")}, file.UserBackup{Path: "/f", To: "/f.~"}}
	o.Observe(step.Event{Kind: step.Planned, Step: "ssh", Check: true, Changes: plan})
	want := []string{"say: --- /dev/null\n+++ /f\n@@ -0,0 +1 @@\n+a"}
	if !reflect.DeepEqual(out.Lines, want) {
		t.Errorf("lines = %q, want %q", out.Lines, want)
	}
}

func TestMacChangeSaysAnEmptyDiffAsOneEmptyLine(t *testing.T) {
	out := &effecttest.Out{}
	o := linesOf(out, "ws client macos apply")
	plan := []step.Change{file.UserWrite{Path: "/f", Before: []byte("a\n"), After: []byte("a\n"), Exists: true}}
	o.Observe(step.Event{Kind: step.Planned, Step: "ssh", Check: true, Changes: plan})
	if want := []string{"say: "}; !reflect.DeepEqual(out.Lines, want) {
		t.Errorf("lines = %q, want %q", out.Lines, want)
	}
}

func TestSetupWithNoStepRunsThemAll(t *testing.T) {
	m := effecttest.NewMachine()
	writeConfs(m, "", values.Text("WS_USER", "ada"))
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = m
	status, _, stderr := runWs(p, "client", "macos", "apply")
	if status == 2 || strings.Contains(stderr, "Usage") {
		t.Errorf("status = %d, stderr = %q; want the steps run, not the usage", status, stderr)
	}
}

func TestSetupStepNamesCompleteOnly(t *testing.T) {
	c := newMacApplyCmd(fakeProcess(), new(string))
	got, _ := c.GetFlagCompletionFunc("only")
	names, _ := got(c, nil, "")
	if len(names) != 0 {
		t.Errorf("--only completes %q; want nothing, as no step is left", names)
	}
}

func TestMacStateRefusesRootAndArgs(t *testing.T) {
	p := fakeProcess()
	p.Euid = 0
	if status, _, stderr := runWs(p, "client", "macos", "state"); status != 1 ||
		!strings.HasPrefix(stderr, "ws client macos state: run it as yourself, not root") {
		t.Errorf("as root: status = %d, stderr = %q", status, stderr)
	}
	for _, args := range [][]string{{"state", "extra"}, {"state", "--check"}} {
		if status, _, stderr := runWs(fakeProcess(), append([]string{"client", "macos"}, args...)...); status != 2 ||
			!strings.Contains(stderr, "--help' for usage.") {
			t.Errorf("%q: status = %d, stderr = %q", args, status, stderr)
		}
	}
	if status, _, stderr := runWs(fakeProcess(), "client", "macos", "state"); status != 1 || !strings.Contains(stderr, "no server in") {
		t.Errorf("no server: status = %d, stderr = %q", status, stderr)
	}
}

func TestMacStateHelp(t *testing.T) {
	status, stdout, _ := runWs(fakeProcess(), "client", "macos", "state", "-h")
	for _, want := range []string{"--json", "applies nothing"} {
		if status != 0 || !strings.Contains(stdout, want) {
			t.Errorf("status = %d, help = %q, want it to hold %q", status, stdout, want)
		}
	}
}

func TestMacStateWritesReports(t *testing.T) {
	var b bytes.Buffer
	reports := []step.Report{
		{Step: "git", Changes: []step.Pending{{Text: "write ~/.gitconfig\n+a\n"}}},
		{Step: "zed", After: "git"},
	}
	if err := writeMacState(&b, []serverDoc{{Steps: reports}}, false); err != nil {
		t.Fatal(err)
	}
	want := "  git\n    pending: write ~/.gitconfig\n  zed\n    not read: after git\n"
	if b.String() != want {
		t.Errorf("got %q, want %q", b.String(), want)
	}
	b.Reset()
	if err := engine.WriteJSON(&b, macDoc{Version: "v1", Steps: reports[1:]}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"version": "v1"`) || !strings.Contains(b.String(), `"after": "git"`) {
		t.Errorf("json = %s", b.String())
	}
}

func TestMacStateReadsMacConf(t *testing.T) {
	m := effecttest.NewMachine()
	writeConfs(m, "", values.Text("WS_USER", "ada"))
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = m
	status, stdout, _ := runWs(p, "client", "macos", "state", "--json")
	if status != 0 || !strings.Contains(stdout, `"version": "v1.2.3"`) || strings.Contains(stdout, `"step"`) {
		t.Errorf("status = %d, stdout = %q", status, stdout)
	}
}

// macStateProcess returns a process with a Mac that has a values.yml.
func macStateProcess() enginecli.Env {
	m := effecttest.NewMachine()
	writeConfs(m, "", values.Text("WS_USER", "ada"))
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = m
	return p
}

func TestMacReadSaysThatMacConfIsMissing(t *testing.T) {
	m := effecttest.NewMachine()
	writeConfs(m, "", "")
	delete(m.Files, "/etc/ws/values.yml")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = m
	status, _, stderr := runWs(p, "client", "macos", "state")
	if status == 0 || !strings.Contains(stderr, "is missing") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestMacStatePrintsTextOrJSON(t *testing.T) {
	for _, args := range [][]string{{"state"}, {"state", "--json"}} {
		status, stdout, stderr := runWs(macStateProcess(), append([]string{"client", "macos"}, args...)...)
		if status != 0 || (args[len(args)-1] == "--json" && stdout == "") || stderr != "" {
			t.Errorf("%v: status = %d, stdout = %q, stderr = %q", args, status, stdout, stderr)
		}
	}
}
