package engine

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

const (
	enterScreen = "\x1b[?1049h"
	leaveScreen = "\x1b[?1049l"
)

// applyCall returns a call whose stdin types keys, and the buffers that it prints on. A caller
// that sets TTY and ErrTTY has it draw a screen on stderr.
func applyCall(keys io.Reader) (Call, *bytes.Buffer, *bytes.Buffer) {
	var out, drawn bytes.Buffer
	return Call{
		Path: "ws x apply", In: keys, Out: &out, Err: &drawn, Machine: effecttest.NewMachine(),
		Getenv: func(k string) string { return map[string]string{"TERM": "xterm"}[k] },
	}, &out, &drawn
}

// targetOf returns a Target whose run does do, with no steps to name.
func targetOf(do func(context.Context, Effects, step.Options) (string, error)) Target {
	return Target{Prepare: func(_ context.Context, c Call) (Run, error) {
		return Run{M: c.Machine, Groups: func(bool) []tui.Group { return []tui.Group{tui.NewGroup("", nil)} }, Do: do}, nil
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

func TestApplyDrawsTheScreenOnTheScreensEffects(t *testing.T) {
	c, out, drawn := applyCall(strings.NewReader(""))
	c.TTY, c.ErrTTY = true, true
	c.Selected = true
	c.Machine = local.OSMachine{}
	var m effect.Machine
	var o effect.Out
	err := Apply(t.Context(), c, targetOf(func(_ context.Context, e Effects, _ step.Options) (string, error) {
		m, o = e.M, e.Out
		_, err := io.WriteString(e.Text, "the key\n")
		if out.Len() != 0 {
			t.Error("the text reached stdout under the screen")
		}
		return "past the screen", err
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := o.(*tui.Bridge); !ok {
		t.Errorf("the run's steps got %T, want the screen's bridge", o)
	}
	if lm, ok := m.(local.OSMachine); !ok || lm.Ticket == nil || lm.Stdin != local.NoStdin || lm.Stderr == nil {
		t.Errorf("the run's machine is %+v, want one that keeps its commands off the terminal", m)
	}
	if out.String() != "the key\n" || strings.Contains(drawn.String(), "the key") {
		t.Errorf("stdout = %q, drawn = %q", out.String(), drawn.String())
	}
	left := strings.LastIndex(drawn.String(), leaveScreen)
	if !strings.Contains(drawn.String(), enterScreen) || left < 0 || !strings.HasSuffix(drawn.String()[left:], "ws x apply: past the screen\n") {
		t.Errorf("stderr = %q, want a whole screen and the line past it", drawn.String())
	}
}

func TestApplyStopsPastTheCurrentStep(t *testing.T) {
	keys, press := io.Pipe()
	t.Cleanup(func() { _ = press.Close() })
	c, _, _ := applyCall(keys)
	c.TTY, c.ErrTTY = true, true
	c.Selected = true
	err := Apply(t.Context(), c, targetOf(func(_ context.Context, _ Effects, o step.Options) (string, error) {
		_, _ = io.WriteString(press, "\x03\r")
		for o.Stop() == nil {
			time.Sleep(time.Millisecond)
		}
		return "", o.Stop()
	}))
	if !errors.Is(err, SignalExit{Sig: syscall.SIGINT}) {
		t.Errorf("err = %v", err)
	}
}

func TestApplyOpensNoListWithCheck(t *testing.T) {
	c, _, _ := applyCall(strings.NewReader("q"))
	c.TTY, c.ErrTTY = true, true
	c.Check = true
	var checks []bool
	err := Apply(t.Context(), c, targetOf(func(_ context.Context, _ Effects, o step.Options) (string, error) {
		checks = append(checks, o.Check)
		return "", nil
	}))
	if err != nil || len(checks) != 1 || !checks[0] {
		t.Errorf("err = %v, runs with check = %v, want one", err, checks)
	}
}

// The server's run takes a machine that ws_home confines, apart from the call's: the screen's run
// and the list's scan must act on it.
func TestApplyActsOnTheRunsMachineNotTheCalls(t *testing.T) {
	for _, selected := range []bool{true, false} {
		keys, press := io.Pipe()
		t.Cleanup(func() { _ = press.Close() })
		c, _, _ := applyCall(keys)
		c.TTY, c.ErrTTY, c.Selected = true, true, selected
		confined := effecttest.NewMachine()
		var ran []effect.Machine
		err := Apply(t.Context(), c, Target{Prepare: func(context.Context, Call) (Run, error) {
			return Run{M: confined, Groups: func(bool) []tui.Group { return []tui.Group{tui.NewGroup("", nil)} },
				Do: func(_ context.Context, e Effects, o step.Options) (string, error) {
					ran = append(ran, e.M)
					if o.Check { // The list's scan: leave the list past it.
						_, _ = io.WriteString(press, "q")
					}
					return "", nil
				}}, nil
		}})
		if err != nil || len(ran) != 1 || ran[0] != confined {
			t.Errorf("selected %v: err = %v, the run acted on %v, want the run's machine alone", selected, err, ran)
		}
	}
}
