package apply

import "github.com/zadykian/ws/step"

// eventNames is the name of each kind of event in --json's lines.
var eventNames = map[step.Kind]string{
	step.Planned: "planned",
	step.Started: "started",
	step.Done:    "done",
	step.Failed:  "failed",
	step.Skipped: "skipped",
}

// EventLine is one event of a run, as ws server apply --json prints it.
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
		l.Changes = append(l.Changes, step.PendingOf(c))
	}
	if e.Err != nil {
		l.Error = e.Err.Error()
	}
	return l
}
