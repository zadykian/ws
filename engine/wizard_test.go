package engine

import (
	"context"
	"errors"
	"io"
	"slices"
	"syscall"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

// testTarget is a Target whose wizard has stages that need no machine: no forms and no state read.
func testTarget(list ...step.Step) Target {
	return Target{Wizard: Side{
		WsRun: func(list []step.Step) (WsRun, bool) {
			for _, s := range list {
				if run, ok := s.(wsStage); ok {
					return WsRun{Name: run.Name()}, true
				}
			}
			return WsRun{}, false
		},
		Stages: func(WizardEffects, WizardOptions) []step.Step { return list },
	}}
}

// tick is the change that tickStep plans.
type tick string

func (c tick) String() string { return string(c) }

// tickStep is a stage that plans a change and records that it applied.
type tickStep struct {
	name    string
	applied *bool
}

func (s tickStep) Name() string    { return s.name }
func (s tickStep) WaitsOn() string { return "" }
func (s tickStep) Plan(context.Context) (step.Plan, error) {
	return step.Plan{Changes: []step.Change{tick("x")}}, nil
}
func (s tickStep) Apply(context.Context, step.Plan) error {
	if s.applied != nil {
		*s.applied = true
	}
	return nil
}

// wsStage is the stage that testTarget takes for ws's run.
type wsStage struct{ tickStep }

func TestWizardConfigEndsACutAsAPlainRunDoes(t *testing.T) {
	c, _, _ := applyCall(nil)
	err := wizardRun{c: c, t: testTarget()}.config().Cut(syscall.SIGINT)
	var cut SignalExit
	if err == nil || errors.As(err, &cut) {
		t.Errorf("err = %v, want one of status 1", err)
	}
}

// A tick in ws's run whole asks for a run, though the last one passed. Another stage's asks for none.
func TestWizardRunGoesOnceMoreForATick(t *testing.T) {
	w := wizardRun{t: testTarget(wsStage{tickStep{name: "ws's run"}}, tickStep{name: "one"})}
	run, _ := w.t.Wizard.WsRun(w.plainStages())
	if !w.optionsOf(tui.QueueChoice{Whole: []string{"ws's run"}}, run.Name).Update {
		t.Error("a tick in ws's run left the last run to speak")
	}
	if w.optionsOf(tui.QueueChoice{Whole: []string{"one"}}, run.Name).Update {
		t.Error("another stage's tick ran ws again")
	}
}

func TestWizardWorkLeavesOutWhatTheChoiceDoes(t *testing.T) {
	c, _, _ := applyCall(nil)
	var applied bool
	w := wizardRun{c: c, t: testTarget(tickStep{name: "one", applied: &applied})}
	o := WizardOptions{Left: func(string, string) bool { return true }}
	cfg := w.config()
	cfg.Text = io.Discard
	screen := tui.NewRunScreen("ws", cfg.Theme, tui.Group{})
	if err := tui.Run(t.Context(), cfg, screen, w.work(o, nil)); err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Error("a stage left out applied")
	}
}

func TestWizardScanOfASideWithNoRunReportsNoRoles(t *testing.T) {
	c, _, _ := applyCall(nil)
	var applied bool
	w := wizardRun{c: c, t: testTarget(tickStep{name: "one", applied: &applied})}
	cfg := w.config()
	cfg.Text = io.Discard
	screen := tui.NewRunScreen("ws", cfg.Theme, tui.Group{})
	if err := tui.Run(t.Context(), cfg, screen, w.scan(nil)); err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Error("a scan applied a stage")
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
