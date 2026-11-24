package cli

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestCommandLineToolsPlanNothingWhereTheyStand(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["xcode-select -p"] = effecttest.Result{Out: []byte("/Library/Developer/CommandLineTools\n")}
	changes, err := commandLineTools{M: m}.Plan(t.Context())
	if err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v", changes, err)
	}
}

func TestCommandLineToolsAskForTheInstallAndWaitUntilTheyStand(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["xcode-select -p"] = effecttest.Result{Err: errors.New("none")}
	m.Results["xcode-select --install"] = effecttest.Result{}
	m.After = map[string]map[string]effecttest.Result{
		"xcode-select --install": {"xcode-select -p": {Out: []byte("/Library/Developer/CommandLineTools\n")}}}
	var out effecttest.Out
	tools := commandLineTools{M: m, Out: &out, Poll: time.Millisecond}
	changes, err := tools.Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v; want one", changes, err)
	}
	if err := changes[0].(installTools).Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "waiting for the command line tools") {
		t.Errorf("said %q", out.Lines)
	}
	if after, err := tools.Plan(t.Context()); err != nil || len(after) != 0 {
		t.Errorf("a plan after the wait found %v, %v", after, err)
	}
}

func TestCommandLineToolsWaitEndsWhereTheContextEnds(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["xcode-select -p"] = effecttest.Result{Err: errors.New("none")}
	m.Results["xcode-select --install"] = effecttest.Result{Err: errors.New("a request stands")}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var out effecttest.Out
	err := installTools{commandLineTools{M: m, Out: &out, Poll: time.Hour}}.Apply(ctx)
	if err == nil || !strings.Contains(err.Error(), "stopped before the command line tools stood") ||
		!strings.Contains(err.Error(), "a request stands") {
		t.Errorf("err = %v, want the stop with the request's failure", err)
	}
}

// withoutTools has a look for the command line tools fail once, and the tools stand after it.
func withoutTools(m *effecttest.Machine) {
	missing := effecttest.Result{Err: errors.New("xcode-select: error: unable to get active developer directory")}
	m.Results[toolsLine] = missing
	m.After = map[string]map[string]effecttest.Result{}
	m.After[toolsLine] = map[string]effecttest.Result{toolsLine: {Out: []byte("/Library/Developer/CommandLineTools\n")}}
}

func TestSetupInstallsTheCommandLineToolsBeforeAnyGit(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	withoutTools(m)
	m.Results["xcode-select --install"] = effecttest.Result{}
	status, stderr := runSetup(p, rel)
	if status != 0 || !strings.Contains(stderr, "waiting for the command line tools") {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	ask, git := indexOf(m, "run xcode-select --install"), indexOf(m, "run git")
	if ask < 0 || git < ask {
		t.Errorf("the install is at %d, git at %d: %q", ask, git, m.Calls)
	}
	if slices.Index(m.Calls[ask:], "run "+toolsLine) < 0 {
		t.Errorf("setup did not look for the tools after it asked: %q", m.Calls)
	}
}

func TestSetupLeavesTheCommandLineToolsThatStand(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	if status, stderr := runSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if indexOf(m, "run xcode-select --install") >= 0 {
		t.Errorf("the install was asked for: %q", m.Calls)
	}
}

func TestSetupPlanSaysTheCommandLineToolsAreDueAndGoesNoFurther(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	withoutTools(m)
	status, stderr := runSetup(p, rel, "--plan")
	if status != 2 || !strings.Contains(stderr, "would install the command line tools") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if indexOf(m, "run git") >= 0 || indexOf(m, "run xcode-select --install") >= 0 {
		t.Errorf("the plan ran git or asked for the tools: %q", m.Calls)
	}
	leftNothing(t, m)
}
