package cmd

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos"
)

func TestSetupCommand(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.machine = m
	status, _, stderr := runWs(p, "", "client", "macos", "apply", "--only", "ssh", "--bogus")
	if status != 2 || !strings.Contains(stderr, "ws client macos apply: unknown flag: --bogus") {
		t.Errorf("status = %d, stderr = %q; want a usage error", status, stderr)
	}
	p = fakeProcess()
	p.machine = m
	if status, _, stderr = runWs(p, "", "client", "macos", "apply", "--only", "ssh"); status != 1 || !strings.Contains(stderr, "is missing") {
		t.Errorf("without HOME: status = %d, stderr = %q; want a missing mac.conf", status, stderr)
	}
}

func TestSetupRefusesRoot(t *testing.T) {
	p := fakeProcess()
	p.euid = 0
	if status, _, stderr := runWs(p, "", "client", "macos", "apply", "--only", "ssh"); status != 1 ||
		!strings.HasPrefix(stderr, "ws client macos apply: run it as yourself, not root") {
		t.Errorf("as root: status = %d, stderr = %q", status, stderr)
	}
}

func TestProgOutPrefixesLines(t *testing.T) {
	out := progOut{Out: &effecttest.Out{}, prog: "ws client macos apply", seen: map[string]bool{}}
	out.Say("wrote %s", "f")
	out.Say("%s: %s", "Answers", "bad")
	out.Warn("w %d", 1)
	out.Warn("w %d", 1)
	want := []string{
		"say: ws client macos apply: wrote f", "say: ws client macos apply: Answers: bad", "say: ws client macos apply: warning: w 1",
	}
	if got := out.Out.(*effecttest.Out).Lines; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestProgOutObservesAPlanAndWhatAStepIsBehind(t *testing.T) {
	out := progOut{Out: &effecttest.Out{}, prog: "ws client macos apply", seen: map[string]bool{}}
	plan := []step.Change{
		macos.Write{Path: "/f", After: []byte("a\n")}, macos.Backup{Path: "/f", To: "/f.~"}, stringChange("would run x"),
	}
	out.Observe(step.Event{Kind: step.Planned, Step: "ssh", Changes: plan})
	out.Observe(step.Event{Kind: step.Started, Step: "ssh", Changes: plan})
	out.Observe(step.Event{Kind: step.Planned, Step: "ssh", Check: true, Changes: plan})
	out.Observe(step.Event{Kind: step.Skipped, Step: "git", After: "ssh"})
	out.Observe(step.Event{Kind: step.Skipped, Step: "zed"})
	out.Observe(step.Event{Kind: step.Planned, Step: "ssh", Check: true, Changes: plan, LeftOut: true})
	out.Observe(step.Event{Kind: step.Skipped, Step: "git", After: "ssh", LeftOut: true})
	out.Observe(step.Event{Kind: step.Skipped, Step: "cld", Err: errors.New("no file"), LeftOut: true})
	want := []string{
		"say: --- /dev/null\n+++ /f\n@@ -0,0 +1 @@\n+a", "say: ws client macos apply: would run x",
		"say: ws client macos apply: git: after ssh", "say: ws client macos apply: warning: left out: no file",
	}
	if got := out.Out.(*effecttest.Out).Lines; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

type stringChange string

func (c stringChange) String() string { return string(c) }

func TestSetupWithNoStepRunsThemAll(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.machine = m
	status, _, stderr := runWs(p, "", "client", "macos", "apply")
	if status == 2 || strings.Contains(stderr, "Usage") {
		t.Errorf("status = %d, stderr = %q; want the steps run, not the usage", status, stderr)
	}
}

func TestSetupCheckWithNoStepPlansSSHOnce(t *testing.T) {
	m := effecttest.NewMachine()
	keyed(m)
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.machine = m
	_, stdout, stderr := runWs(p, "", "client", "macos", "apply", "--check")
	if n := strings.Count(stdout+stderr, "+++ /h/.ssh/config.d/ws"); n != 1 {
		t.Errorf("the config.d/ws diff prints %d times; want 1\n%s%s", n, stdout, stderr)
	}
}

func TestSetupStepNamesCompleteOnly(t *testing.T) {
	c := newMacApplyCmd(fakeProcess())
	got, _ := c.GetFlagCompletionFunc("only")
	names, _ := got(c, nil, "")
	if want := []string{"ssh", "cable", "git", "ghostty", "zed", "cld", "zsh", "tunnel", "networks"}; !reflect.DeepEqual(names, want) {
		t.Errorf("--only completes %q; want %q", names, want)
	}
}

func TestMacStateRefusesRootAndArgs(t *testing.T) {
	p := fakeProcess()
	p.euid = 0
	if status, _, stderr := runWs(p, "", "client", "macos", "state"); status != 1 ||
		!strings.HasPrefix(stderr, "ws client macos state: run it as yourself, not root") {
		t.Errorf("as root: status = %d, stderr = %q", status, stderr)
	}
	for _, args := range [][]string{{"state", "extra"}, {"state", "--check"}} {
		if status, _, stderr := runWs(fakeProcess(), "", append([]string{"client", "macos"}, args...)...); status != 2 ||
			!strings.Contains(stderr, "--help' for usage.") {
			t.Errorf("%q: status = %d, stderr = %q", args, status, stderr)
		}
	}
	if status, _, stderr := runWs(fakeProcess(), "", "client", "macos", "state"); status != 1 || !strings.Contains(stderr, "is missing") {
		t.Errorf("no mac.conf: status = %d, stderr = %q", status, stderr)
	}
}

func TestMacStateHelp(t *testing.T) {
	status, stdout, _ := runWs(fakeProcess(), "", "client", "macos", "state", "-h")
	for _, want := range []string{"--json", "applies nothing", "a question"} {
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
	if err := writeMacState(&b, reports); err != nil {
		t.Fatal(err)
	}
	want := "  git\n    pending: write ~/.gitconfig\n  zed\n    not read: after git\n"
	if b.String() != want {
		t.Errorf("got %q, want %q", b.String(), want)
	}
	b.Reset()
	if err := writeJSON(&b, macDoc{Version: "v1", Steps: reports[1:]}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"version": "v1"`) || !strings.Contains(b.String(), `"after": "git"`) {
		t.Errorf("json = %s", b.String())
	}
}

func TestMacStateReadsMacConf(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.machine = m
	status, stdout, _ := runWs(p, "", "client", "macos", "state", "--json")
	if status != 0 || !strings.Contains(stdout, `"version": "v1.2.3"`) || !strings.Contains(stdout, `"steps"`) ||
		!strings.Contains(stdout, `"kind": "package"`) {
		t.Errorf("status = %d, stdout = %q", status, stdout)
	}
}

// altScreen is what a screen writes first: the switch to the alternate screen.
const altScreen = "\x1b[?1049h"

// macStateAtTerminal returns a process that stands at a terminal, with a Mac that has a mac.conf.
func macStateAtTerminal() process {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h", "TERM": "xterm-256color"})
	p.machine = m
	p.tty, p.errTTY, p.outTTY = true, true, true
	return p
}

func TestMacReadSaysThatMacConfIsMissing(t *testing.T) {
	_, err := macRead(fakeProcess())(t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "is missing") {
		t.Errorf("err = %v", err)
	}
}

func TestMacStateDrawsTheScreenAtATerminal(t *testing.T) {
	status, stdout, stderr := runWs(macStateAtTerminal(), "q", "client", "macos", "state")
	if status != 0 || stdout != "" || !strings.Contains(stderr, altScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacStatePrintsTextWithPlainOrJSON(t *testing.T) {
	for _, flag := range []string{"--plain", "--json"} {
		status, stdout, stderr := runWs(macStateAtTerminal(), "", "client", "macos", "state", flag)
		if status != 0 || stdout == "" || strings.Contains(stderr, altScreen) {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q", flag, status, stdout, stderr)
		}
	}
}

func TestMacStatePrintsTextToAPipeOrAFile(t *testing.T) {
	p := macStateAtTerminal()
	p.outTTY = false
	status, stdout, stderr := runWs(p, "", "client", "macos", "state")
	if status != 0 || stdout == "" || strings.Contains(stderr, altScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}
