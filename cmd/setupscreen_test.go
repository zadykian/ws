package cmd

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
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
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
	status, stdout, stderr := runWs(p, "", "client", "macos", "apply", "--check", "--only", "ssh")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacApplyReportsAFailurePastTheScreen(t *testing.T) {
	p, m := macTerminal()
	delete(m.Results, "mktemp -d")
	status, _, stderr := runWs(p, "", "client", "macos", "apply", "--check", "--only", "ssh")
	left := strings.LastIndex(stderr, leaveScreen)
	if status != 1 || left < 0 || !strings.Contains(stderr[left:], "ws client macos apply: ") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestMacApplyDrawsNothingWithPlain(t *testing.T) {
	p, _ := macTerminal()
	_, wantOut, wantErr := runWs(withEnv(p, map[string]string{"HOME": "/h", "TERM": ""}), "", "client", "macos", "apply", "--check", "--only", "ssh")
	status, stdout, stderr := runWs(p, "", "client", "macos", "apply", "--plain", "--check", "--only", "ssh")
	if status != 0 || stdout != wantOut || stderr != wantErr || strings.Contains(stderr, "\x1b") {
		t.Errorf("status = %d, stdout = %q, stderr = %q, want the lines %q", status, stdout, stderr, wantErr)
	}
}

func TestSetupOnScreenRunsOnTheScreensEffects(t *testing.T) {
	p, _ := macTerminal()
	p.machine = effect.OSMachine{}
	c, _ := screenCommand(t, "")
	var machines []effect.Machine
	var outs []effect.Out
	run := func(_ context.Context, m effect.Machine, out effect.Out, _ io.Writer, _ step.Options) error {
		machines, outs = append(machines, m), append(outs, out)
		return nil
	}
	if err := setupOnScreen(c, p, tui.NewGroup("", nil), run, step.Options{Check: true}); err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 {
		t.Fatalf("the run was made %d times, want once", len(outs))
	}
	if _, ok := outs[0].(*tui.Bridge); !ok {
		t.Errorf("the run's steps got %T, want the screen's bridge", outs[0])
	}
	m, ok := machines[0].(effect.OSMachine)
	if !ok || m.Ticket == nil || m.Stdin != effect.NoStdin || m.Stderr == nil {
		t.Errorf("the run's machine is %+v, want one that keeps its commands off the terminal", machines[0])
	}
}

func TestSetupOnScreenPrintsWhatStaysOnStdoutPastTheScreen(t *testing.T) {
	p, _ := macTerminal()
	c, drawn := screenCommand(t, "")
	var stdout bytes.Buffer
	c.SetOut(&stdout)
	run := func(_ context.Context, _ effect.Machine, _ effect.Out, text io.Writer, _ step.Options) error {
		_, err := io.WriteString(text, "the key\n")
		if stdout.Len() != 0 {
			t.Error("the text reached stdout under the screen")
		}
		return err
	}
	if err := setupOnScreen(c, p, tui.NewGroup("", nil), run, step.Options{Check: true}); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "the key\n" || strings.Contains(drawn.String(), "the key") {
		t.Errorf("stdout = %q, drawn = %q", stdout.String(), drawn.String())
	}
}

func TestSetupOnScreenStopsPastTheCurrentStep(t *testing.T) {
	p, _ := macTerminal()
	c, _ := screenCommand(t, "")
	keys, press := io.Pipe()
	t.Cleanup(func() { _ = press.Close() })
	c.SetIn(keys)
	var reached bool
	run := func(ctx context.Context, _ effect.Machine, _ effect.Out, _ io.Writer, o step.Options) error {
		steps := []step.Step{
			stepOf{"one", func(context.Context) error {
				_, _ = io.WriteString(press, "\x03\r")
				for o.Stop() == nil {
					time.Sleep(time.Millisecond)
				}
				return nil
			}},
			stepOf{"two", func(context.Context) error { reached = true; return nil }},
		}
		_, err := step.RunWith(ctx, steps, o)
		return err
	}
	err := setupOnScreen(c, p, tui.NewGroup("", nil), run, step.Options{Check: true})
	if !errors.Is(err, signalExit{syscall.SIGINT}) || reached {
		t.Errorf("err = %v, the next step reached = %v", err, reached)
	}
}
