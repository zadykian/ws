package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

func TestWizardWorkRunsTheStagesOnTheScreensEffects(t *testing.T) {
	p, m := macTerminal()
	c, _ := screenCommand(t, "")
	w := wizardRun{c, p, macSide(c, p), wizardOptions{}}
	cfg := w.config()
	screen := tui.NewRunScreen("ws", cfg.Theme, tui.Group{})
	var text bytes.Buffer
	cfg.Text = &text
	if err := tui.Run(context.Background(), cfg, screen, w.work(wizardOptions{check: true}, &wizard.Sudo{})); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "run caffeinate") }) {
		t.Errorf("no run began: calls = %q", m.Calls)
	}
}

func TestWizardConfigEndsACutAsAPlainRunDoes(t *testing.T) {
	p, _ := macTerminal()
	c, _ := screenCommand(t, "")
	err := wizardRun{c, p, macSide(c, p), wizardOptions{}}.config().Cut(syscall.SIGINT)
	var cut signalExit
	if err == nil || errors.As(err, &cut) {
		t.Errorf("err = %v, want one of status 1", err)
	}
}

func TestWizardCheckAtATerminalGoesStraightToTheRun(t *testing.T) {
	p, m := macTerminal()
	_, stdout, stderr := runWs(p, "", "mac", "wizard", "--check")
	if stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("stdout = %q, stderr = %q", stdout, stderr)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "run caffeinate") }) {
		t.Errorf("no run began: calls = %q", m.Calls)
	}
}

func TestWizardPlainDrawsNoScreenAtATerminal(t *testing.T) {
	p, _ := macTerminal()
	_, _, stderr := runWs(p, "", "mac", "wizard", "--plain", "--check")
	if strings.Contains(stderr, enterScreen) {
		t.Errorf("--plain drew: %q", stderr)
	}
}

func TestWizardOpensTheListsAndLeavesWithQ(t *testing.T) {
	p, m := macTerminal()
	status, stdout, stderr := runWs(p, "q", "mac", "wizard")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if slices.Contains(m.Calls, "run xcode-select --install") {
		t.Errorf("a stage applied: calls = %q", m.Calls)
	}
}

func TestWizardCutAtTheListsEndsWithStatusOne(t *testing.T) {
	p, _ := macTerminal()
	var out, errs bytes.Buffer
	root := newRoot(p, newMacCmd(p), newServerCmd(p))
	status := execute(context.Background(), root, []string{"mac", "wizard"}, repeating("\x03"), &out, &errs)
	if status != 1 {
		t.Errorf("status = %d, want the plain wizard's 1: %q", status, errs.String())
	}
}

// freed is a machine whose rm frees the lock, for the run that a scan's end lets in.
type freed struct{ *effecttest.Machine }

func (f freed) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "rm" {
		delete(f.Dirs, args[len(args)-1])
	}
	return f.Machine.Run(ctx, name, args...)
}

func TestWizardRunsTheStagesChosenAlone(t *testing.T) {
	p, m := macTerminal()
	m.Results["xcode-select -p"] = effecttest.Result{Out: []byte("/x\n")}
	m.Files["/x/usr/bin/git"], m.Modes["/x/usr/bin/git"] = nil, 0o755
	p.machine = freed{m}
	var out bytes.Buffer
	root := newRoot(p, newMacCmd(p), newServerCmd(p))
	keys := newAfterScan("n \r")
	status := execute(context.Background(), root, []string{"mac", "wizard"}, keys, &out, keys)
	if status != 0 || strings.Count(strings.Join(m.Calls, "\n"), "run caffeinate") != 2 {
		t.Fatalf("status = %d, want a scan and a run: calls = %q", status, m.Calls)
	}
	if strings.Contains(strings.Join(m.Calls, "\n"), "run curl") {
		t.Errorf("Homebrew applied, left out: calls = %q", m.Calls)
	}
}

func TestWsRunOfFindsNoneWhereNoneIs(t *testing.T) {
	if run := wsRunOf([]step.Step{stepOf{name: "one"}}); run != nil {
		t.Errorf("wsRunOf = %v, want nil", run)
	}
}

// A tick in ws's run whole asks for a run, though the last one passed. Another stage's asks for none.
func TestWizardRunGoesOnceMoreForATick(t *testing.T) {
	w := wizardRun{p: fakeProcess(), side: macSide(nil, fakeProcess())}
	run := wsRunOf(w.plainStages()).Name()
	if !w.optionsOf(tui.QueueChoice{Whole: []string{"ws's run"}}, run).update {
		t.Error("a tick in ws's run left the last run to speak")
	}
	if w.optionsOf(tui.QueueChoice{Whole: []string{"ws mac apply"}}, run).update {
		t.Error("another stage's tick ran ws again")
	}
}

func TestWizardWorkLeavesOutWhatTheChoiceDoes(t *testing.T) {
	p, m := macTerminal()
	c, _ := screenCommand(t, "")
	w := wizardRun{c, p, macSide(c, p), wizardOptions{}}
	o := wizardOptions{left: func(string, string) bool { return true }}
	cfg := w.config()
	cfg.Text = io.Discard
	screen := tui.NewRunScreen("ws", cfg.Theme, tui.Group{})
	if err := tui.Run(context.Background(), cfg, screen, w.work(o, &wizard.Sudo{})); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(m.Calls, "run xcode-select --install") {
		t.Errorf("a stage left out applied: calls = %q", m.Calls)
	}
}

func TestWizardScanChangesNothing(t *testing.T) {
	p, m := macTerminal()
	c, _ := screenCommand(t, "")
	w := wizardRun{c, p, macSide(c, p), wizardOptions{}}
	cfg := w.config()
	screen := tui.NewRunScreen("ws", cfg.Theme, tui.Group{})
	if err := tui.Run(context.Background(), cfg, screen, w.scan(&wizard.Sudo{})); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "run xcode-select -p") || slices.Contains(m.Calls, "run xcode-select --install") {
		t.Errorf("calls = %q, want a check of the tools alone", m.Calls)
	}
}

func TestAfterStageWaitsForTheStageItself(t *testing.T) {
	var heard, called []string
	a := afterStage{observer(func(e step.Event) { heard = append(heard, e.Step) }), "one", func() {
		called = append(called, heard[len(heard)-1])
	}}
	a.Observe(step.Event{Step: "one", Kind: step.Started})
	a.Observe(step.Event{Step: "one", Kind: step.Skipped})
	a.Observe(step.Event{Parent: "two", Step: "one", Kind: step.Done})
	a.Observe(step.Event{Step: "two", Kind: step.Done})
	if len(called) != 0 {
		t.Fatalf("then ran early: %q", called)
	}
	a.Observe(step.Event{Step: "one", Kind: step.Done})
	if !slices.Equal(called, []string{"one"}) {
		t.Errorf("then = %q, want it once, past the event", called)
	}
}

// observer hears a run's events through a function.
type observer func(step.Event)

func (f observer) Observe(e step.Event) { f(e) }
