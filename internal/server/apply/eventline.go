package apply

import (
	"github.com/zadykian/ws/step"
)

// eventNames is the name of each kind of event in --json's lines.
var eventNames = map[step.Kind]string{
	step.Planned: "planned",
	step.Started: "started",
	step.Done:    "done",
	step.Failed:  "failed",
	step.Skipped: "skipped",
}

// EventLine is one event of a run, as ws server apply --json prints it and the Mac's wizard reads
// it. Each of Changes holds its first line alone, as firstLine says, and an error that names a
// change holds the same.
type EventLine struct {
	Event   string         `json:"event"`
	Parent  string         `json:"parent,omitempty"`
	Step    string         `json:"step"`
	Index   int            `json:"index"`
	Check   bool           `json:"check,omitempty"`
	Changes []step.Pending `json:"changes,omitempty"`
	LeftOut bool           `json:"left_out,omitempty"`
	After   string         `json:"after,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// NewEventLine turns an event into data: a Change holds what it applies with, and a line crosses a
// wire.
func NewEventLine(e step.Event) EventLine {
	l := EventLine{
		Event: eventNames[e.Kind], Parent: e.Parent, Step: e.Step, Index: e.Index,
		Check: e.Check, LeftOut: e.LeftOut, After: e.After,
	}
	for _, c := range e.Changes {
		l.Changes = append(l.Changes, firstLine(c))
	}
	if e.Err != nil {
		l.Error = e.Err.Error()
	}
	return l
}

// firstLine is a change as a line holds it: what changes. The rest of its text is a file's diff, with
// what the file holds now, and the Mac's wizard runs ws in a unit, whose journal keeps each line past
// the run and cuts one of more than 48 KiB. ws server state --json has the diffs.
func firstLine(c step.Change) step.Pending {
	return step.Pending{Kind: step.KindOf(c), Text: step.Headline(c)}
}
