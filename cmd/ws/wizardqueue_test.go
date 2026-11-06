package main

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
)

func queueOf(p process) []tui.Stage {
	return wizardQueue(p, wizardStages(wizardEffects{p: p, hand: &wizard.Hand{}, sudo: &wizard.Sudo{}}, wizardOptions{}))
}

func stageOf(t *testing.T, queue []tui.Stage, name string) tui.Stage {
	t.Helper()
	at := slices.IndexFunc(queue, func(s tui.Stage) bool { return s.Name == name })
	if at < 0 {
		t.Fatalf("queue has no stage %q", name)
	}
	return queue[at]
}

func TestWizardQueueSetsEachStageOnItsMachine(t *testing.T) {
	queue := queueOf(fakeProcess())
	on := map[tui.Side]int{}
	for _, s := range queue {
		on[s.Side]++
	}
	if on[tui.MacSide] != 11 || on[tui.WsSide] != 4 {
		t.Errorf("sides = %v, want 11 on the Mac and 4 on ws", on)
	}
	if run := stageOf(t, queue, "ws's run"); run.Side != tui.WsSide || run.Waits == "" {
		t.Errorf("ws's run = %+v, want it on ws, behind the install", run)
	}
}

// A chosen Cable locks ws mac apply, whose steps it applies: left out, the cable's service stays unset.
func TestWizardQueueHoldsWsMacApplyBesideCable(t *testing.T) {
	if cable := stageOf(t, queueOf(fakeProcess()), "Cable"); cable.Waits != "ws mac apply" {
		t.Errorf("Cable = %+v, want it behind ws mac apply", cable)
	}
}

func TestWizardQueueNestsWhatItsStagesRun(t *testing.T) {
	p := fakeProcess()
	queue := queueOf(p)
	setup := stageOf(t, queue, "ws mac apply").Nested
	if len(setup) != 1 || setup[0].Name != "ws mac apply" || !slices.Equal(setup[0].Steps, setupStepNames(p)) {
		t.Errorf("ws mac apply holds %+v, want its steps by its name", setup)
	}
	run := stageOf(t, queue, "ws's run").Nested
	if len(run) != len(roles.List()) || run[0].Name != roles.List()[0].Name || len(run[0].Steps) != 0 {
		t.Errorf("ws's run holds %+v, want each role unread", run)
	}
}
