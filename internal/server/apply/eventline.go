package apply

import (
	"encoding/json"
	"errors"
	"strings"

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

// ParseEvent reads a line of a run's log, and says whether it is an event. The commands of a run write
// to that log too, and some write JSON, so it takes a line with EventLine's fields alone, and a kind.
func ParseEvent(line string) (step.Event, bool) {
	var l EventLine
	dec := json.NewDecoder(strings.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&l); err != nil || dec.More() {
		return step.Event{}, false
	}
	kind, ok := kindNamed(l.Event)
	if !ok || l.Step == "" {
		return step.Event{}, false
	}
	e := step.Event{
		Kind: kind, Parent: l.Parent, Step: l.Step, Index: l.Index,
		Check: l.Check, LeftOut: l.LeftOut, After: l.After,
	}
	for _, p := range l.Changes {
		e.Changes = append(e.Changes, lineChange{p})
	}
	if l.Error != "" {
		e.Err = errors.New(l.Error)
	}
	return e, true
}

// kindNamed is the kind of event that a line names.
func kindNamed(name string) (step.Kind, bool) {
	for k, n := range eventNames {
		if n == name {
			return k, true
		}
	}
	return 0, false
}

// lineChange is a change as a line named it: a screen shows it, and nothing applies it.
type lineChange struct{ p step.Pending }

func (c lineChange) String() string { return c.p.Text }

// Kind is the kind that the line gave.
func (c lineChange) Kind() step.ChangeKind { return c.p.Kind }
