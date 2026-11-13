package tui

import (
	"github.com/zadykian/ws/engine/core/step"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// reportedRow is the step of a group, as a report says it stands. What the step would change and
// what stops it are the words of its row, as a report has no output to keep.
func reportedRow(parent string, index int, report step.Report) pickRow {
	row := stepRow{parent: parent, index: index, name: report.Step, state: stepSame}
	picked := pickRow{stepRow: row, after: report.After}
	for _, change := range report.Changes {
		picked.state = stepChanged
		picked.said = append(picked.said, widget.Line{Kind: widget.Said, Text: change.Text})
	}
	for _, item := range report.Items {
		if item.Kind == step.CheckItem {
			picked.state = stepFailed
			picked.said = append(picked.said, widget.Line{Kind: widget.Warned, Text: item.Conflict})
		}
	}
	if report.After != "" {
		picked.state = stepSkipped
	}
	picked.chosen = picked.pending()
	return picked
}
