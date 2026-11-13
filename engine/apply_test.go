package engine

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// applyCall returns a call whose stdin types keys, and the buffers it prints on.
func applyCall(keys io.Reader) (Call, *bytes.Buffer, *bytes.Buffer) {
	var out, drawn bytes.Buffer
	return Call{
		Path: "ws x apply", In: keys, Out: &out, Err: &drawn, Machine: effecttest.NewMachine(),
		Getenv: func(k string) string { return map[string]string{"TERM": "xterm"}[k] },
	}, &out, &drawn
}

// targetOf returns a Target whose run does do.
func targetOf(do func(context.Context, Effects, step.Options) (string, error)) Target {
	return Target{Prepare: func(_ context.Context, c Call) (Run, error) {
		return Run{M: c.Machine, Do: do}, nil
	}}
}

func TestApplyPlainSaysTheLineAfterTheRun(t *testing.T) {
	c, _, drawn := applyCall(strings.NewReader(""))
	err := Apply(t.Context(), c, targetOf(func(_ context.Context, e Effects, _ step.Options) (string, error) {
		e.Out.Say("working")
		return "all done", nil
	}))
	if want := "ws x apply: working\nws x apply: all done\n"; err != nil || drawn.String() != want {
		t.Errorf("err = %v, stderr = %q, want %q", err, drawn.String(), want)
	}
}

func TestApplyPlainReturnsTheRunsErrorAndSaysNoLine(t *testing.T) {
	c, _, drawn := applyCall(strings.NewReader(""))
	boom := errors.New("boom")
	err := Apply(t.Context(), c, targetOf(func(context.Context, Effects, step.Options) (string, error) { return "line", boom }))
	if !errors.Is(err, boom) || drawn.Len() != 0 {
		t.Errorf("err = %v, stderr = %q", err, drawn.String())
	}
}

func TestApplyJSONPrintsTheEventsOnStdout(t *testing.T) {
	c, out, _ := applyCall(strings.NewReader(""))
	c.TTY, c.ErrTTY, c.JSON = true, true, true
	err := Apply(t.Context(), c, targetOf(func(_ context.Context, _ Effects, o step.Options) (string, error) {
		o.Observer.Observe(step.Event{Kind: step.Started, Step: "one"})
		return "", nil
	}))
	if err != nil || !strings.Contains(out.String(), `"event":"started"`) {
		t.Errorf("err = %v, stdout = %q", err, out.String())
	}
}
