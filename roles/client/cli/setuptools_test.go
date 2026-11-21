package cli

import (
	"context"
	"errors"
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
