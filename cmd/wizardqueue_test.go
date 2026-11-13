package cmd

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/roles/client/macos/setup"
	"github.com/zadykian/ws/roles/server/ubuntu"
)

func queueOf(p process) []tui.Stage {
	return wizardQueue(wizardStages(wizardEffects{p: p, hand: &wizard.Hand{}, sudo: &wizard.Sudo{}}, wizardOptions{}))
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
	if len(nested) != 1 || nested[0].Name != "ws client macos apply" || !slices.Equal(nested[0].Steps, setup.Names()) {
		t.Errorf("ws client macos apply holds %+v, want its steps by its name", nested)
	}
	run := stageOf(t, queue, "ws's run").Nested
	if len(run) != len(ubuntu.Names()) || run[0].Name != ubuntu.Names()[0] || len(run[0].Steps) != 0 {
		t.Errorf("ws's run holds %+v, want each role unread", run)
	}
}

func TestWizardLeftFollowsTheChoice(t *testing.T) {
	const setup = "ws client macos apply"
	choice := tui.QueueChoice{Only: []string{setup, "Tools"}, Partial: map[string][]string{setup: {setup + "/ssh", setup + "/zed"}}}
	left := wizardLeft(choice)
	for _, tc := range []struct {
		parent, name string
		want         bool
	}{
		{"", "Tools", false},
		{"", "Homebrew", true},
		{"", setup, false},
		{setup, "ssh", false},
		{setup, "git", true},
	} {
		if got := left(tc.parent, tc.name); got != tc.want {
			t.Errorf("left(%q, %q) = %v, want %v", tc.parent, tc.name, got, tc.want)
		}
	}
	if whole := wizardLeft(tui.QueueChoice{Only: []string{setup}}); whole(setup, "git") {
		t.Error("a stage run whole left a step out")
	}
	if off := wizardLeft(tui.QueueChoice{Only: []string{"Tools"}}); !off(setup, "git") {
		t.Error("a stage wholly off ran a step")
	}
}

func TestReportRolesLeavesPrepareOut(t *testing.T) {
	var sections []tui.StateSection
	var said []string
	report := func(stage string, s []tui.StateSection, problem string) {
		sections, said = s, append(said, stage, problem)
	}
	states := []role.RoleState{
		{Role: role.Prepare, Steps: []step.Report{{Step: "apt"}}},
		{Role: "fans", Steps: []step.Report{{Step: "unit"}}},
	}
	reportRoles("ws's run", states, nil, report)
	if len(sections) != 1 || sections[0].Name != "fans" || !slices.Equal(said, []string{"ws's run", ""}) {
		t.Errorf("sections = %+v, said %q; want the role fans alone", sections, said)
	}
}

func TestReportRolesSaysWhyItHasNone(t *testing.T) {
	var problem string
	report := func(_ string, s []tui.StateSection, why string) {
		if s != nil {
			t.Errorf("sections = %+v, want none", s)
		}
		problem = why
	}
	reportRoles("ws's run", nil, errors.New("ws's report refused"), report)
	if problem != "ws's report refused" {
		t.Errorf("problem = %q", problem)
	}
}
