package cli

import (
	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/wizard"
	"slices"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
)

func queueOf(p enginecli.Env) []tui.Stage {
	return engine.WizardQueue(macSide(p, new(string)), wizardStages(wizardEffects{p: p, hand: &wizard.Hand{}, sudo: &host.Sudo{}}, engine.WizardOptions{}, "ws"))
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
	on := map[string]int{}
	for _, s := range queue {
		on[s.Side]++
	}
	if on[string(stage.Mac)] != 11 || on[string(session.SideOf("ws"))] != 4 {
		t.Errorf("sides = %v, want 11 on the Mac and 4 on ws", on)
	}
	if run := stageOf(t, queue, "ws's run"); run.Side != string(session.SideOf("ws")) || run.Waits == "" {
		t.Errorf("ws's run = %+v, want it on ws, behind the install", run)
	}
}

// The Mac's pane stands left of ws's as the stages first name them, so the first stage is the Mac's.
func TestWizardQueueNamesTheMacFirst(t *testing.T) {
	if queue := queueOf(fakeProcess()); queue[0].Side != string(stage.Mac) {
		t.Errorf("the first stage is on %q, want %q", queue[0].Side, stage.Mac)
	}
}

// A chosen Cable locks ws client macos apply, whose steps it applies: left out, the cable's service
// stays unset.
func TestWizardQueueHoldsWsMacApplyBesideCable(t *testing.T) {
	if cable := stageOf(t, queueOf(fakeProcess()), "Cable"); cable.Waits != "ws client macos apply" {
		t.Errorf("Cable = %+v, want it behind ws client macos apply", cable)
	}
}

func TestWizardQueueNestsWhatItsStagesRun(t *testing.T) {
	queue := queueOf(fakeProcess())
	nested := stageOf(t, queue, "ws client macos apply").Nested
	if len(nested) != 1 || nested[0].Name != "ws client macos apply" || !slices.Equal(nested[0].Steps, macos.Names()) {
		t.Errorf("ws client macos apply holds %+v, want its steps by its name", nested)
	}
	run := stageOf(t, queue, "ws's run").Nested
	want := ubuntucli.NestedRows((&stages.WsRun{}).Name())
	same := func(a, b tui.Group) bool { return a.Name == b.Name && slices.Equal(a.Steps, b.Steps) }
	if len(want) == 0 || !slices.EqualFunc(run, want, same) {
		t.Errorf("ws's run holds %+v, want the server cli's rows %+v", run, want)
	}
}
