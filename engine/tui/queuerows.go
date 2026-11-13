package tui

import (
	"github.com/zadykian/ws/engine/core/step"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// reportedRow is the step of a group, as a report says it stands. What the step would change and
// what stops it are the words of its row, as a report has no output to keep.
func reportedRow(parent string, index int, report step.Report) PickRow {
	row := StepRow{Parent: parent, Index: index, Name: report.Step, State: StepSame}
	picked := PickRow{StepRow: row, After: report.After}
	for _, change := range report.Changes {
		picked.State = StepChanged
		picked.Said = append(picked.Said, widget.Line{Kind: widget.Said, Text: change.Text})
	}
	for _, item := range report.Items {
		if item.Kind == step.CheckItem {
			picked.State = StepFailed
			picked.Said = append(picked.Said, widget.Line{Kind: widget.Warned, Text: item.Conflict})
		}
	}
	if report.After != "" {
		picked.State = StepSkipped
	}
	picked.Chosen = picked.Pending()
	return picked
}
