package main

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

// repeating types keys again and again, so that one lands once the list takes it.
type repeating string

func (r repeating) Read(b []byte) (int, error) {
	time.Sleep(5 * time.Millisecond)
	return copy(b, r), nil
}

func TestMacApplyOpensTheListAtATerminal(t *testing.T) {
	p, _ := macTerminal()
	status, stdout, stderr := runWs(p, "q", "mac", "apply")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacApplyGoesStraightToTheRunWithAStepOrCheck(t *testing.T) {
	for _, args := range [][]string{{"mac", "apply", "ssh"}, {"mac", "apply", "--check"}} {
		p, m := macTerminal()
		status, _, stderr := runWs(p, "q", args...)
		if !slices.Contains(m.Calls, "run mktemp -d") {
			t.Errorf("%v: status = %d, no run began: calls = %q, stderr = %q", args, status, m.Calls, stderr)
		}
	}
}

func TestServerApplyOpensTheListAndLeavesWithQ(t *testing.T) {
	p, _ := serverTerminal()
	status, stdout, stderr := runWs(p, "q", "server", "apply")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if strings.Contains(stderr, "changed:") {
		t.Errorf("a run began: %q", stderr)
	}
}

func TestPickSetupReturnsTheStepsChosen(t *testing.T) {
	p, _ := macTerminal()
	c, _ := screenCommand(t, "")
	c.SetIn(repeating("n \r"))
	var checks []bool
	build := func(check bool) setupBuild {
		return func(_ effect.Machine, _ effect.Out, _ io.Writer) []step.Step {
			checks = append(checks, check)
			return []step.Step{
				stepOf{"one", func(context.Context) error { return nil }},
				stepOf{"two", func(context.Context) error { return nil }},
			}
		}
	}
	o, ok, err := pickSetup(c, p, build)
	if err != nil || !ok || o.Check || o.LeftOut == nil {
		t.Fatalf("options = %+v, ok = %v, err = %v", o, ok, err)
	}
	if o.LeftOut("", "one") || !o.LeftOut("", "two") {
		t.Error("the choice leaves out a step it chose, or keeps one it didn't")
	}
	if o.LeftOut("", "ssh again") {
		t.Error("the choice leaves out a step that no row offers")
	}
	if slices.Contains(checks, false) {
		t.Errorf("the list built steps without a check: %v", checks)
	}
}

func TestServerApplyChecksFromTheListAndSaysSo(t *testing.T) {
	p, m := serverTerminal()
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n tmux"] = effecttest.Result{Out: []byte("tmux ii \n")}
	var errs bytes.Buffer
	root := newRoot(p, newMacCmd(p), newServerCmd(p))
	// A scan holds c back, so each turn ends by emptying the filter that the next one types.
	status := execute(t.Context(), root, []string{"server", "apply"}, repeating("/tmux\rac/\x7f\x7f\x7f\x7f\r"), io.Discard, &errs)
	if want := "ws server apply: would change: none\n"; status != 0 || !strings.HasSuffix(errs.String(), want) {
		t.Errorf("status = %d, stderr ends %q, want %q", status, errs.String()[max(0, errs.Len()-300):], want)
	}
}
