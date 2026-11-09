package main

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"sync"
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

// afterScan is a terminal that holds keys back until the list draws its count in place of the scan's.
// Keys of one read can straddle the scan's end, and those past it then act on a list that missed the
// first. It reports a size first, as a screen with none draws nothing. A list that never draws the
// count gets the keys all the same, and the test fails on what they did.
type afterScan struct {
	keys  repeating
	sized sync.Once
	mu    sync.Mutex
	drawn bytes.Buffer
	shown sync.Once
	ended chan struct{}
}

func newAfterScan(keys repeating) *afterScan {
	return &afterScan{keys: keys, ended: make(chan struct{})}
}

func (s *afterScan) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.drawn.Write(b)
	if bytes.Contains(s.drawn.Bytes(), []byte(" chosen")) {
		s.shown.Do(func() { close(s.ended) })
	}
	return len(b), nil
}

func (s *afterScan) Read(b []byte) (int, error) {
	size := 0
	s.sized.Do(func() { size = copy(b, "\x1b[8;24;80t") })
	if size > 0 {
		return size, nil
	}
	select {
	case <-s.ended:
	case <-time.After(10 * time.Second):
	}
	return s.keys.Read(b)
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
	keys := newAfterScan("n \r")
	c.SetIn(keys)
	c.SetErr(keys)
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
