package tui

import (
	"slices"

	"github.com/zadykian/ws/step"
)

// learned returns the list with a row for e's step, where a stage runs it and no scan named it. A
// new Mac has no mac.conf to name a stage's steps by, and no role of ws has steps before ws is there.
func (l pickList) learned(e step.Event) pickList {
	if e.Parent == "" || l.place(e) >= 0 {
		return l
	}
	row := pickRow{stepRow: stepRow{parent: e.Parent, index: e.Index, name: e.Step}}
	at, replaced := -1, 0
	for i, r := range l.rows {
		switch {
		case r.number > 0 && r.name == e.Parent:
			row.side, row.stage, at = r.side, r.name, i+1
		case r.stage != "" && r.parent == e.Parent:
			row.side, row.stage, at = r.side, r.stage, i+1
		case r.stage != "" && r.index == unreadIndex && r.name == e.Parent:
			row.side, row.stage, at, replaced = r.side, r.stage, i, 1
		}
	}
	if at < 0 {
		return l
	}
	l.rows = slices.Replace(slices.Clone(l.rows), at, at+replaced, row)
	return l
}

// reported returns the list with stage's rows as sections say, in place of those it had.
func (l pickList) reported(stage string, sections []StateSection) pickList {
	head := slices.IndexFunc(l.rows, func(r pickRow) bool { return r.number > 0 && r.name == stage })
	if head < 0 {
		return l
	}
	rows := slices.Clone(l.rows[:head+1])
	for _, section := range sections {
		for i, report := range section.Reports {
			rows = append(rows, l.rows[head].reportRow(section.Name, i, report))
		}
	}
	for _, r := range l.rows[head+1:] {
		if r.stage != stage {
			rows = append(rows, r)
		}
	}
	l.rows = rows
	return l.settled()
}

// reportRow is a step of the stage r, as a report says it stands. What the step would change and
// what stops it are the words of its row, as a report has no output to keep.
func (r pickRow) reportRow(parent string, index int, report step.Report) pickRow {
	row := stepRow{parent: parent, index: index, name: report.Step, state: stepSame}
	picked := pickRow{stepRow: row, side: r.side, stage: r.name, after: report.After}
	for _, change := range report.Changes {
		picked.state = stepChanged
		picked.said = append(picked.said, Line{Said, change.Text})
	}
	for _, item := range report.Items {
		if item.Kind == step.CheckItem {
			picked.state = stepFailed
			picked.said = append(picked.said, Line{Warned, item.Conflict})
		}
	}
	if report.After != "" {
		picked.state = stepSkipped
	}
	picked.chosen = picked.pending()
	return picked
}

// follow returns the list with how a run left e's step. A step that starts takes the cursor, so that
// its list shows it. The boxes stand, as a run takes no choice.
func (l pickList) follow(e step.Event) pickList {
	l = l.learned(e)
	place := l.place(e)
	if place < 0 {
		return l
	}
	l.rows = slices.Clone(l.rows)
	l.rows[place].state, l.rows[place].after = stateAfter(e), e.After
	if e.Kind == step.Planned || e.Kind == step.Started {
		l = l.facing(l.rows[place].side)
		l.at = cursor(max(slices.Index(l.lines(), pickLine{row: place}), 0))
	}
	return l
}
