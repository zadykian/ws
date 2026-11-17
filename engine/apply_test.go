package engine

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// applyCall returns a call and the buffers that it prints on.
func applyCall() (Call, *bytes.Buffer, *bytes.Buffer) {
	var out, said bytes.Buffer
	return Call{
		Path: "ws x apply", Out: &out, Err: &said, Machine: effecttest.NewMachine(),
		Getenv: func(string) string { return "" },
	}, &out, &said
}

// targetOf returns a Target whose run does do, with no steps to name.
func targetOf(do func(context.Context, Effects, step.Options) (string, error)) Target {
	return Target{Prepare: func(_ context.Context, c Call) (Run, error) {
		return Run{M: c.Machine, Do: do}, nil
	}}
}

func TestApplyPlainSaysTheLineAfterTheRun(t *testing.T) {
	c, _, said := applyCall()
	err := Apply(t.Context(), c, targetOf(func(_ context.Context, e Effects, _ step.Options) (string, error) {
		e.Out.Say("working")
		return "all done", nil
	}))
	if want := "ws x apply: working\nws x apply: all done\n"; err != nil || said.String() != want {
		t.Errorf("err = %v, stderr = %q, want %q", err, said.String(), want)
	}
}

func TestApplyPlainReturnsTheRunsErrorAndSaysNoLine(t *testing.T) {
	c, _, said := applyCall()
	boom := errors.New("boom")
	err := Apply(t.Context(), c, targetOf(func(context.Context, Effects, step.Options) (string, error) { return "line", boom }))
	if !errors.Is(err, boom) || said.Len() != 0 {
		t.Errorf("err = %v, stderr = %q", err, said.String())
	}
}

func TestApplyJSONPrintsTheEventsOnStdout(t *testing.T) {
	c, out, _ := applyCall()
	c.JSON = true
	err := Apply(t.Context(), c, targetOf(func(_ context.Context, _ Effects, o step.Options) (string, error) {
		o.Observer.Observe(step.Event{Kind: step.Started, Step: "one"})
		return "", nil
	}))
	if err != nil || !strings.Contains(out.String(), `"event":"started"`) {
		t.Errorf("err = %v, stdout = %q", err, out.String())
	}
}

// The server's run takes a machine that WS_HOME confines, apart from the call's: the run must act
// on it.
func TestApplyActsOnTheRunsMachineNotTheCalls(t *testing.T) {
	c, _, _ := applyCall()
	confined := effecttest.NewMachine()
	var ran []effect.Machine
	err := Apply(t.Context(), c, Target{Prepare: func(context.Context, Call) (Run, error) {
		return Run{M: confined, Do: func(_ context.Context, e Effects, _ step.Options) (string, error) {
			ran = append(ran, e.M)
			return "", nil
		}}, nil
	}})
	if err != nil || len(ran) != 1 || ran[0] != confined {
		t.Errorf("err = %v, the run acted on %v, want the run's machine alone", err, ran)
	}
}
