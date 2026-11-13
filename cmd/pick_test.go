package cmd

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
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
	status, stdout, stderr := runWs(p, "q", "client", "macos", "apply")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacApplyGoesStraightToTheRunWithAStepOrCheck(t *testing.T) {
	for _, args := range [][]string{{"client", "macos", "apply", "--only", "ssh"}, {"client", "macos", "apply", "--check"}} {
		p, m := macTerminal()
		status, _, stderr := runWs(p, "q", args...)
		if !slices.Contains(m.Calls, "run mktemp -d") {
			t.Errorf("%v: status = %d, no run began: calls = %q, stderr = %q", args, status, m.Calls, stderr)
		}
	}
}

func TestApplyRunsTheStepsChosenFromTheList(t *testing.T) {
	keys := newAfterScan("n \r")
	steps := []step.Step{
		stepOf{"one", func(context.Context) error { return nil }},
		stepOf{"two", func(context.Context) error { return nil }},
	}
	var seen []step.Options
	var run engine.Run
	run = engine.Run{
		M:      effecttest.NewMachine(),
		Groups: func(bool) []tui.Group { return []tui.Group{tui.NewGroup("", steps)} },
		Do: func(ctx context.Context, _ engine.Effects, o step.Options) (string, error) {
			seen = append(seen, o)
			if !o.Check { // The steps' plans never settle, so only a check can run them.
				return "", nil
			}
			_, err := step.RunWith(ctx, steps, o)
			return "", err
		},
		Choose: func(choice tui.Choice) (engine.Run, step.Options, error) {
			left := func(_, name string) bool { return !slices.Contains(choice.Only, name) }
			return run, step.Options{Check: choice.Check, LeftOut: left}, nil
		},
	}
	call := engine.Call{
		Path: "ws x apply", In: keys, Out: io.Discard, Err: keys, Machine: run.M,
		Getenv: func(k string) string { return map[string]string{"TERM": "xterm"}[k] }, TTY: true, ErrTTY: true,
	}
	err := engine.Apply(t.Context(), call, engine.Target{Prepare: func(context.Context, engine.Call) (engine.Run, error) { return run, nil }})
	if err != nil || len(seen) < 2 {
		t.Fatalf("err = %v, runs = %d", err, len(seen))
	}
	o := seen[len(seen)-1]
	if o.Check || o.LeftOut == nil || o.LeftOut("", "one") || !o.LeftOut("", "two") {
		t.Errorf("the run's options %+v leave out a step it chose, or keep one it didn't", o)
	}
	for _, o := range seen[:len(seen)-1] {
		if !o.Check {
			t.Error("the list scanned without a check")
		}
	}
}
