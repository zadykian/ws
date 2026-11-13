package engine

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

func TestWizardLeftFollowsTheChoice(t *testing.T) {
	const setup = "ws client macos apply"
	choice := tui.QueueChoice{Only: []string{setup, "Tools"}, Partial: map[string][]string{setup: {setup + "/ssh", setup + "/zed"}}}
	left := WizardLeft(choice)
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
	if whole := WizardLeft(tui.QueueChoice{Only: []string{setup}}); whole(setup, "git") {
		t.Error("a stage run whole left a step out")
	}
	if off := WizardLeft(tui.QueueChoice{Only: []string{"Tools"}}); !off(setup, "git") {
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
	ReportRoles("ws's run", states, nil, report)
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
	ReportRoles("ws's run", nil, errors.New("ws's report refused"), report)
	if problem != "ws's report refused" {
		t.Errorf("problem = %q", problem)
	}
}
