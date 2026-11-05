package step

import "context"

// ItemKind says what an Item is. The values are those of --json.
type ItemKind string

// The kinds of item.
const (
	PackageItem ItemKind = "package"
	ServiceItem ItemKind = "service"
	FileItem    ItemKind = "file"
	LinkItem    ItemKind = "link"
	DirItem     ItemKind = "directory"
)

// Item is a part of what a step manages, as a report shows it. It holds no secret, as --check
// shows none.
type Item struct {
	Kind ItemKind `json:"kind"`
	Name string   `json:"name"`
	// Want is the state ws sets and Found the state on the machine, as a person reads them.
	Want  string `json:"want"`
	Found string `json:"found"`
	// Conflict says why Found stands against ws, where it does.
	Conflict string `json:"conflict,omitempty"`
	// Diff is a file's diff to what ws writes, empty for a secret.
	Diff string `json:"diff,omitempty"`
}

// Reporter is a Step, or a part of one, that says what it manages. A step without one shows its
// plan alone.
type Reporter interface {
	// Report reads alone, as Plan does.
	Report(ctx context.Context) ([]Item, error)
}

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
	Items   []Item    `json:"items,omitempty"`
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
		r, err := read(ctx, s, e)
		if err != nil {
			return reports, err
		}
		reports = append(reports, r)
	}
	return reports, nil
}

func read(ctx context.Context, s Step, e Event) (Report, error) {
	r := Report{Step: e.Step, After: e.After}
	for _, c := range e.Changes {
		r.Changes = append(r.Changes, Pending{KindOf(c), c.String()})
	}
	reporter, ok := s.(Reporter)
	if !ok || e.Kind == Skipped {
		return r, nil
	}
	var err error
	r.Items, err = reporter.Report(ctx)
	return r, err
}
