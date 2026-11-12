package tui

import (
	"testing"

	"github.com/zadykian/ws/step"
)

func TestAReportSaysHowItsStepStands(t *testing.T) {
	for _, tc := range []struct {
		report step.Report
		state  stepState
		chosen bool
	}{
		{step.Report{Step: "a"}, stepSame, false},
		{step.Report{Step: "a", Changes: []step.Pending{{Text: "sets it"}}}, stepChanged, true},
		{step.Report{Step: "a", After: "b"}, stepSkipped, true},
		{step.Report{Step: "a", Items: []step.Item{{Kind: step.CheckItem, Conflict: "no"}}}, stepFailed, false},
	} {
		got := reportedRow("base", 2, tc.report)
		if got.state != tc.state || got.chosen != tc.chosen || got.parent != "base" || got.index != 2 {
			t.Errorf("%+v gives %+v", tc.report, got)
		}
	}
}
