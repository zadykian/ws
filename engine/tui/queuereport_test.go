package tui

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

func TestAReportSaysHowItsStepStands(t *testing.T) {
	for _, tc := range []struct {
		report step.Report
		state  StepState
		chosen bool
	}{
		{step.Report{Step: "a"}, StepSame, false},
		{step.Report{Step: "a", Changes: []step.Pending{{Text: "sets it"}}}, StepChanged, true},
		{step.Report{Step: "a", After: "b"}, StepSkipped, true},
		{step.Report{Step: "a", Items: []step.Item{{Kind: step.CheckItem, Conflict: "no"}}}, StepFailed, false},
	} {
		got := reportedRow("base", 2, tc.report)
		if got.State != tc.state || got.Chosen != tc.chosen || got.Parent != "base" || got.Index != 2 {
			t.Errorf("%+v gives %+v", tc.report, got)
		}
	}
}
