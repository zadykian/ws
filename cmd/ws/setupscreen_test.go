package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/step"
)

// keyed gives the Mac a mac.conf that names the file key it has for ws's login, as ssh's step needs.
func keyed(m *effecttest.Machine) {
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\nWS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub\n")
	m.Files["/h/.ssh/id_ed25519.pub"] = []byte("ssh-ed25519 AAAA ada\n")
	m.Results["ssh-keygen -l -f /h/.ssh/id_ed25519.pub"] = effecttest.Result{}
}

// macTerminal returns a process with a terminal, as the Mac's user, with a mac.conf and what ssh's
// step looks at.
func macTerminal() (process, *effecttest.Machine) {
	m := effecttest.NewMachine()
	keyed(m)
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h", "TERM": "xterm"})
	p.machine, p.tty, p.errTTY = m, true, true
	return p, m
}

func TestMacApplyAtATerminalDrawsAWholeScreen(t *testing.T) {
	p, _ := macTerminal()
	status, stdout, stderr := runWs(p, "", "mac", "apply", "--check", "ssh")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacApplyReportsAFailurePastTheScreen(t *testing.T) {
	p, m := macTerminal()
	delete(m.Results, "mktemp -d")
	status, _, stderr := runWs(p, "", "mac", "apply", "--check", "ssh")
	left := strings.LastIndex(stderr, leaveScreen)
	if status != 1 || left < 0 || !strings.Contains(stderr[left:], "ws mac apply: ") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestMacApplyDrawsNothingWithPlain(t *testing.T) {
	p, _ := macTerminal()
	_, wantOut, wantErr := runWs(withEnv(p, map[string]string{"HOME": "/h", "TERM": ""}), "", "mac", "apply", "--check", "ssh")
	status, stdout, stderr := runWs(p, "", "mac", "apply", "--plain", "--check", "ssh")
	if status != 0 || stdout != wantOut || stderr != wantErr || strings.Contains(stderr, "\x1b") {
		t.Errorf("status = %d, stdout = %q, stderr = %q, want the lines %q", status, stdout, stderr, wantErr)
	}
}

// ending is a stepOf with what a run leaves for its end.
type ending struct {
	stepOf
	finish func(context.Context) error
}

func (e ending) Finish(ctx context.Context) error { return e.finish(ctx) }

func TestRunSetupFinishesNoStepLeftOut(t *testing.T) {
	finish := func(context.Context) error { return errors.New("a step left out finished") }
	steps := []step.Step{ending{stepOf{"one", func(context.Context) error { return nil }}, finish}}
	left := func(_, _ string) bool { return true }
	if err := runSetup(t.Context(), steps, step.Options{LeftOut: left}); err != nil {
		t.Error(err)
	}
}

func TestSetupOnScreenRunsOnTheScreensEffects(t *testing.T) {
	p, _ := macTerminal()
	p.machine = effect.OSMachine{}
	c, _ := screenCommand(t, "")
	var machines []effect.Machine
	var outs []effect.Out
	build := func(m effect.Machine, out effect.Out, _ io.Writer) []step.Step {
		machines, outs = append(machines, m), append(outs, out)
		return []step.Step{stepOf{"one", func(context.Context) error { return nil }}}
	}
	if err := setupOnScreen(c, p, build, step.Options{Check: true}); err != nil {
		t.Fatal(err)
	}
	if len(outs) != 2 || outs[0] != nil {
		t.Fatalf("the steps were made on %v, want no effects first, for their names", outs)
	}
	if _, ok := outs[1].(*tui.Bridge); !ok {
		t.Errorf("the run's steps got %T, want the screen's bridge", outs[1])
	}
	m, ok := machines[1].(effect.OSMachine)
	if !ok || m.Ticket == nil || m.Stdin != effect.NoStdin || m.Stderr == nil {
		t.Errorf("the run's machine is %+v, want one that keeps its commands off the terminal", machines[1])
	}
}

func TestSetupOnScreenPrintsWhatStaysOnStdoutPastTheScreen(t *testing.T) {
	p, _ := macTerminal()
	c, drawn := screenCommand(t, "")
	var stdout bytes.Buffer
	c.SetOut(&stdout)
	build := func(_ effect.Machine, _ effect.Out, text io.Writer) []step.Step {
		return []step.Step{ending{stepOf{"one", func(context.Context) error { return nil }}, func(context.Context) error {
			_, err := io.WriteString(text, "the key\n")
			if stdout.Len() != 0 {
				t.Error("the text reached stdout under the screen")
			}
			return err
		}}}
	}
	if err := setupOnScreen(c, p, build, step.Options{Check: true}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "the key\n" || strings.Contains(drawn.String(), "the key") {
		t.Errorf("stdout = %q, drawn = %q", stdout.String(), drawn.String())
	}
}

func TestSetupOnScreenStopsPastTheCurrentStep(t *testing.T) {
	p, _ := macTerminal()
	c, _ := screenCommand(t, "\x03\r")
	var reached, finished bool
	build := func(_ effect.Machine, out effect.Out, _ io.Writer) []step.Step {
		return []step.Step{
			ending{stepOf{"one", func(ctx context.Context) error {
				for stopped := out.(*tui.Bridge).Stop(ctx); stopped() == nil; {
					time.Sleep(time.Millisecond)
				}
				return nil
			}}, func(context.Context) error { finished = true; return nil }},
			stepOf{"two", func(context.Context) error { reached = true; return nil }},
		}
	}
	err := setupOnScreen(c, p, build, step.Options{Check: true})
	if !errors.Is(err, signalExit{syscall.SIGINT}) || reached || finished {
		t.Errorf("err = %v, the next step reached = %v, finished = %v", err, reached, finished)
	}
}
