package step

import (
	"context"
	"strings"
)

// ItemKind says what an Item is. The values are those of --json.
type ItemKind string

// The kinds of item.
const (
	PackageItem ItemKind = "package"
	ServiceItem ItemKind = "service"
	FileItem    ItemKind = "file"
	LinkItem    ItemKind = "link"
	DirItem     ItemKind = "directory"
	// CheckItem is what stops a run: Read makes one of a step's failure.
	CheckItem ItemKind = "check"
)

// Item is a part of what a step manages, as a report shows it. It holds no secret.
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
	// Report reads alone, as Plan does, and follows a Plan that passed: what Plan refuses stays unread.
	Report(ctx context.Context) ([]Item, error)
}

// Itemized is a Change that stands as an Item. A role may build a resource as it plans, where no
// Report reaches it: the change then says what the resource's Report would, and Read shows that.
type Itemized interface {
	Item() Item
}

// Pending is a Change as data: a Change holds what it applies with, and a report crosses a wire.
type Pending struct {
	Kind ChangeKind `json:"kind"`
	Text string     `json:"text"`
}

// PendingOf returns the change as data. Its first line says what changes.
func PendingOf(c Change) Pending {
	text := ExcerptOf(c)
	if t, ok := c.(Titled); ok {
		text = t.Title() + "\n" + text
	}
	return Pending{KindOf(c), text}
}

// Headline is a change's first line, what changes. An error or an event names a change by it: the
// rest can be a file's diff, with what the file holds now, and a unit's journal keeps ws's errors.
func Headline(c Change) string {
	line, _, _ := strings.Cut(PendingOf(c).Text, "\n")
	return line
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
// awaited step has changes to make. A failure stops a run, and so stands as a CheckItem here, and
// Read goes past it: what fails is what a report is for. Only the end of ctx fails Read, which
// returns the reports it has.
func Read(ctx context.Context, steps []Step) ([]Report, error) {
	o := Options{Check: true}
	pending := map[string]bool{}
	reports := make([]Report, 0, len(steps))
	for i, s := range steps {
		if err := ctx.Err(); err != nil {
			return reports, err
		}
		e := o.settle(ctx, s, o.event(i, s), pending)
		if e.Kind == Failed {
			pending[e.Step] = true // what waits on a failed step can't be read either
		}
		reports = append(reports, read(ctx, s, e))
	}
	return reports, nil
}

// read turns a step's last event into its report, with at most one CheckItem. A failed plan stands
// alone: it may be a refusal to read, as host.Plain's of a link, so Report must not go past it.
func read(ctx context.Context, s Step, e Event) Report {
	r := Report{Step: e.Step, After: e.After}
	for _, c := range e.Changes {
		r.Changes = append(r.Changes, PendingOf(c))
	}
	if e.Kind == Skipped {
		return r
	}
	failure := e.Err
	if reporter, ok := s.(Reporter); ok && failure == nil {
		r.Items, failure = reporter.Report(ctx)
	}
	r.Items = append(r.Items, unreported(e.Changes, r.Items)...)
	if failure != nil {
		r.Items = append(r.Items, Item{Kind: CheckItem, Name: e.Step, Want: "passes", Found: "fails",
			Conflict: failure.Error()})
	}
	return r
}

// unreported returns the items of the changes whose resource no item names.
func unreported(changes []Change, items []Item) []Item {
	named := map[string]bool{}
	for _, it := range items {
		named[it.Name] = true
	}
	var rest []Item
	for _, c := range changes {
		if i, ok := c.(Itemized); ok && !named[i.Item().Name] {
			rest = append(rest, i.Item())
		}
	}
	return rest
}
