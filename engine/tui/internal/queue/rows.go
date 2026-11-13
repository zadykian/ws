package queue

import (
	"github.com/zadykian/ws/engine/core/step"

	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// reportedRow is the step of a group, as a report says it stands. What the step would change and
// what stops it are the words of its row, as a report has no output to keep.
func reportedRow(parent string, index int, report step.Report) list.PickRow {
	row := list.StepRow{Parent: parent, Index: index, Name: report.Step, State: list.StepSame}
	picked := list.PickRow{StepRow: row, After: report.After}
	for _, change := range report.Changes {
		picked.State = list.StepChanged
		picked.Said = append(picked.Said, widget.Line{Kind: widget.Said, Text: change.Text})
	}
	for _, item := range report.Items {
		if item.Kind == step.CheckItem {
			picked.State = list.StepFailed
			picked.Said = append(picked.Said, widget.Line{Kind: widget.Warned, Text: item.Conflict})
		}
	}
	if report.After != "" {
		picked.State = list.StepSkipped
	}
	picked.Chosen = picked.Pending()
	return picked
}
