package main

import (
	"encoding/json"
	"io"

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

// eventLine is one event of a run, as ws server apply --json prints it.
type eventLine struct {
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

// newEventLine turns an event into data: a Change holds what it applies with, and a line crosses a
// wire.
func newEventLine(e step.Event) eventLine {
	l := eventLine{
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

// jsonEvents prints each event of a run as a line of JSON, and passes it on to next, which keeps
// ws's own lines on stderr. A write that fails stops the lines and stays in err.
type jsonEvents struct {
	enc  *json.Encoder
	next step.Observer
	err  error
}

func newJSONEvents(w io.Writer, next step.Observer) *jsonEvents {
	return &jsonEvents{enc: json.NewEncoder(w), next: next}
}

// Observe prints the event, then hands it on.
func (j *jsonEvents) Observe(e step.Event) {
	if j.err == nil {
		j.err = j.enc.Encode(newEventLine(e))
	}
	j.next.Observe(e)
}
