package step

import "context"

// Pending is a Change as data: a Change holds what it applies with, and a report crosses a wire.
type Pending struct {
	Kind ChangeKind `json:"kind"`
	Text string     `json:"text"`
}

// Report is what Read finds of one step.
type Report struct {
	Step string `json:"step"`
	// After names the awaited step that kept this one from being read.
	After   string    `json:"after,omitempty"`
	Changes []Pending `json:"changes,omitempty"`
}

// Read reports on each step in order. It goes as a run under check does, so it reads no step whose
// awaited step has changes to make. It stops at the first failure, or at the end of ctx.
func Read(ctx context.Context, steps []Step) ([]Report, error) {
	o := Options{Check: true}
	pending := map[string]bool{}
	reports := make([]Report, 0, len(steps))
	for i, s := range steps {
		if err := ctx.Err(); err != nil {
			return reports, err
		}
		e := o.settle(ctx, s, o.event(i, s), pending)
		if e.Err != nil {
			return reports, e.Err
		}
		reports = append(reports, read(e))
	}
	return reports, nil
}

func read(e Event) Report {
	r := Report{Step: e.Step, After: e.After}
	for _, c := range e.Changes {
		r.Changes = append(r.Changes, Pending{KindOf(c), c.String()})
	}
	return r
}
