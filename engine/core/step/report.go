package step

import (
	"context"
	"errors"
	"strings"
)

// Pending is a Change as data: a Change holds what it applies with, and a read keeps its text alone.
type Pending struct {
	Text string
}

// PendingOf returns the change as data. Its first line says what changes.
func PendingOf(c Change) Pending {
	text := ExcerptOf(c)
	if t, ok := c.(Titled); ok {
		text = t.Title() + "\n" + text
	}
	return Pending{text}
}

// Headline is a change's first line, what changes. An error or an event names a change by it. The
// rest can be a file's diff, with what the file holds now, and a unit's journal keeps ws's errors.
func Headline(c Change) string {
	line, _, _ := strings.Cut(PendingOf(c).Text, "\n")
	return line
}

// Report is what Read finds of one step.
type Report struct {
	Step string
	// After names the awaited step that kept this one from being read.
	After   string
	Changes []Pending
	// Err is why the step failed: its plan's own error, or the wait that settle refused.
	Err error
}

// Read plans each step in order and applies none. It goes as a run under check does, so it plans no
// step whose awaited step has changes to make. A failure stops a run, but Read goes past it: what
// fails is what a read is for. Only the end of ctx fails Read, which returns the reports it has.
func Read(ctx context.Context, steps []Step) ([]Report, error) {
	o := Options{Check: true}
	pending := map[string]bool{}
	reports := make([]Report, 0, len(steps))
	for i, s := range steps {
		if err := ctx.Err(); err != nil {
			return reports, err
		}
		e := o.settle(ctx, s, o.event(i, s), pending)
		r := Report{Step: e.Step, After: e.After}
		for _, c := range e.Changes {
			r.Changes = append(r.Changes, PendingOf(c))
		}
		if e.Kind == Failed {
			pending[e.Step] = true // what waits on a failed step can't be read either
			// settle names the step and says "planning" of a plan that failed, which wraps its error.
			if r.Err = errors.Unwrap(e.Err); r.Err == nil {
				r.Err = e.Err
			}
		}
		reports = append(reports, r)
	}
	return reports, nil
}
