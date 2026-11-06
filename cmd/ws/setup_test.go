package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/step"
)

func TestSetupCommand(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.machine = m
	status, _, stderr := runWs(p, "", "mac", "apply", "ssh", "--bogus")
	if status != 2 || !strings.Contains(stderr, "ws mac apply: unknown flag: --bogus") {
		t.Errorf("status = %d, stderr = %q; want a usage error", status, stderr)
	}
	p = fakeProcess()
	p.machine = m
	if status, _, stderr = runWs(p, "", "mac", "apply", "ssh"); status != 1 || !strings.Contains(stderr, "is missing") {
		t.Errorf("without HOME: status = %d, stderr = %q; want a missing mac.conf", status, stderr)
	}
}

func TestSetupRefusesRoot(t *testing.T) {
	p := fakeProcess()
	p.euid = 0
	if status, _, stderr := runWs(p, "", "mac", "apply", "ssh"); status != 1 ||
		!strings.HasPrefix(stderr, "ws mac apply: run it as yourself, not root") {
		t.Errorf("as root: status = %d, stderr = %q", status, stderr)
	}
}

func TestProgOutPrefixesLines(t *testing.T) {
	out := progOut{Out: &effecttest.Out{}, prog: "ws mac apply", seen: map[string]bool{}}
	out.Say("wrote %s", "f")
	out.Say("%s: %s", "Answers", "bad")
	out.Warn("w %d", 1)
	out.Warn("w %d", 1)
	want := []string{
		"say: ws mac apply: wrote f", "say: ws mac apply: Answers: bad", "say: ws mac apply: warning: w 1",
	}
	if got := out.Out.(*effecttest.Out).Lines; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestProgOutObservesAPlanAndWhatAStepIsBehind(t *testing.T) {
	out := progOut{Out: &effecttest.Out{}, prog: "ws mac apply", seen: map[string]bool{}}
	plan := []step.Change{
		setup.Write{Path: "/f", After: []byte("a\n")}, setup.Backup{Path: "/f", To: "/f.~"}, stringChange("would run x"),
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
		"say: --- /dev/null\n+++ /f\n@@ -0,0 +1 @@\n+a", "say: ws mac apply: would run x",
		"say: ws mac apply: git: after ssh", "say: ws mac apply: warning: left out: no file",
	}
	if got := out.Out.(*effecttest.Out).Lines; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

type stringChange string

func (c stringChange) String() string { return string(c) }

func TestSetupStepsComeInSetupShsOrder(t *testing.T) {
	var names []string
	for _, s := range setupSteps(conf.Conf{}, "/h", "", effecttest.NewMachine(), &effecttest.Out{}, &bytes.Buffer{}, false) {
		names = append(names, s.Name())
	}
	want := []string{"ssh", "cable", "git", "ghostty", "zed", "cld", "tunnel", "networks"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("steps = %q; want %q", names, want)
	}
}

func TestSetupWithNoStepRunsThemAll(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.machine = m
	status, _, stderr := runWs(p, "", "mac", "apply")
	if status == 2 || strings.Contains(stderr, "Usage") {
		t.Errorf("status = %d, stderr = %q; want the steps run, not the usage", status, stderr)
	}
}

func TestSetupCheckWithNoStepPlansSSHOnce(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.machine = m
	_, stdout, stderr := runWs(p, "", "mac", "apply", "--check")
	if n := strings.Count(stdout+stderr, "+++ /h/.ssh/config.d/ws"); n != 1 {
		t.Errorf("the config.d/ws diff prints %d times; want 1\n%s%s", n, stdout, stderr)
	}
}

func TestSetupStepNamesAreTheArguments(t *testing.T) {
	c := newMacApplyCmd(fakeProcess())
	if want := []string{"ssh", "cable", "git", "ghostty", "zed", "cld", "tunnel", "networks"}; !reflect.DeepEqual(c.ValidArgs, want) {
		t.Errorf("ValidArgs = %q; want %q", c.ValidArgs, want)
	}
}
